# Three-State System for Optional Fields

## Overview

The go-contextforge SDK uses a **three-state semantics pattern** for optional fields in API requests. This pattern enables precise control over which fields are included in API calls, supporting three distinct intentions:

1. **Don't update this field** (field omitted from request)
2. **Clear this field** (field explicitly set to zero value)
3. **Set this field to a value** (field set to specific value)

This approach is the **industry standard** used by major Go SDKs including:
- [google/go-github](https://github.com/google/go-github) (500+ dependent repositories)
- [hashicorp/go-tfe](https://github.com/hashicorp/go-tfe) (Terraform Cloud/Enterprise SDK)
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)

The pattern enables **partial updates** where only changed fields are sent to the API, which is essential for:
- Supporting Terraform's `ignore_changes` lifecycle rule
- Avoiding unintended field modifications during updates
- Minimizing network payload size
- Providing clear, type-safe API semantics

## The Three States

### State 1: Field Not Set (nil)

**Intent:** "Don't include this field in the request"

**Implementation:** Pointer or slice is `nil`

**Behavior:** Field is omitted from JSON due to `omitempty` tag

**Example:**
```go
update := &contextforge.ResourceUpdate{
    Name: contextforge.String("new-name"),
    // Description is nil - won't be sent to API
    Description: nil,
    // Tags is nil - won't be sent to API
    Tags: nil,
}
// JSON: {"name": "new-name"}
```

### State 2: Field Set to Zero Value

**Intent:** "Clear this field / set to empty"

**Implementation:**
- Pointer to zero value (`contextforge.String("")`, `contextforge.Int(0)`, `contextforge.Bool(false)`)
- Empty slice (`[]string{}`)

**Behavior:** Field is included in JSON with zero/empty value

**Example:**
```go
update := &contextforge.ResourceUpdate{
    Description: contextforge.String(""),  // Clear description
    Tags: []string{},                       // Clear all tags
}
// JSON: {"description": "", "tags": []}
```

### State 3: Field Set to Value

**Intent:** "Set this field to a specific value"

**Implementation:**
- Pointer to non-zero value
- Populated slice

**Behavior:** Field is included in JSON with the specified value

**Example:**
```go
update := &contextforge.ResourceUpdate{
    Description: contextforge.String("Updated description"),
    Tags: []string{"tag1", "tag2"},
}
// JSON: {"description": "Updated description", "tags": ["tag1", "tag2"]}
```

## How It Works

### JSON Marshaling with omitempty

The `omitempty` JSON tag tells Go's JSON encoder to omit a field from the output if it's "empty". The definition of "empty" varies by type:

| Go Type | Empty Value | Omitted? |
|---------|-------------|----------|
| `*string` (nil) | `nil` | ✅ Yes |
| `*string` (pointer to `""`) | `""` | ❌ No - included as `""` |
| `*int` (nil) | `nil` | ✅ Yes |
| `*int` (pointer to `0`) | `0` | ❌ No - included as `0` |
| `[]string` (nil) | `nil` | ✅ Yes |
| `[]string` (empty) | `[]` | ❌ No - included as `[]` |

**Key Insight:** A nil pointer/slice is considered "empty" and omitted, but a pointer to a zero value or an empty slice is NOT considered empty and will be included.

### Example Marshaling Behavior

```go
type Example struct {
    Field1 *string  `json:"field1,omitempty"`
    Field2 *string  `json:"field2,omitempty"`
    Field3 *string  `json:"field3,omitempty"`
    Tags   []string `json:"tags,omitempty"`
}

ex := Example{
    Field1: nil,                          // State 1: Omitted
    Field2: contextforge.String(""),      // State 2: Included as ""
    Field3: contextforge.String("value"), // State 3: Included as "value"
    Tags:   nil,                          // State 1: Omitted
}

// JSON output:
// {
//   "field2": "",
//   "field3": "value"
// }
```

## Pattern by Field Type

### Scalar Fields (String, Int, Bool, etc.)

**Use pointers with omitempty**

```go
type ResourceUpdate struct {
    Name        *string `json:"name,omitempty"`
    Description *string `json:"description,omitempty"`
    Count       *int    `json:"count,omitempty"`
    Enabled     *bool   `json:"enabled,omitempty"`
}
```

**Why pointers?** Distinguishes between "not set" and "set to zero value":
- `nil` → field not provided
- `&false` → field explicitly set to false
- `&0` → field explicitly set to zero
- `&""` → field explicitly set to empty string

### Array/Slice Fields

**Use direct slices (NOT pointer to slice) with omitempty**

```go
type ResourceUpdate struct {
    Tags                []string `json:"tags,omitempty"`
    AssociatedTools     []string `json:"associatedTools,omitempty"`
    AssociatedResources []string `json:"associatedResources,omitempty"`
}
```

**Why NOT pointer to slice?**
- Following Protocol Buffers convention for repeated fields
- Simpler API (no need to dereference)
- Still supports three states: `nil`, `[]string{}`, `[]string{"value"}`

**Anti-pattern (don't do this):**
```go
// ❌ DON'T USE POINTER TO SLICE
type ResourceUpdate struct {
    Tags *[]string `json:"tags,omitempty"`  // Incorrect!
}
```

### Map Fields

**Use direct maps with omitempty**

```go
type AgentUpdate struct {
    Capabilities map[string]any `json:"capabilities,omitempty"`
    Config       map[string]any `json:"config,omitempty"`
}
```

Maps follow the same pattern as slices:
- `nil` → omitted
- `map[string]any{}` → included as `{}`
- `map[string]any{"key": "value"}` → included with contents

## Create vs Update Semantics

### Create Structs

**Purpose:** Create a new resource

**Pattern:**
- Required fields: direct values (not pointers)
- Optional fields: pointers or slices with omitempty

```go
type ResourceCreate struct {
    // Required fields (direct values)
    URI     string `json:"uri"`
    Name    string `json:"name"`
    Content any    `json:"content"`

    // Optional fields (pointers)
    Description *string  `json:"description,omitempty"`
    MimeType    *string  `json:"mime_type,omitempty"`
    Template    *string  `json:"template,omitempty"`

    // Optional arrays (direct slices)
    Tags []string `json:"tags,omitempty"`
}
```

**Usage:**
```go
// Create with all fields
resource := &contextforge.ResourceCreate{
    URI:         "file:///config.json",         // Required
    Name:        "my-config",                    // Required
    Content:     `{"key": "value"}`,             // Required
    Description: contextforge.String("Config"), // Optional
    Tags:        []string{"config", "json"},     // Optional
}

// Create with minimal fields
resource := &contextforge.ResourceCreate{
    URI:     "file:///config.json",
    Name:    "my-config",
    Content: `{"key": "value"}`,
    // Description and Tags omitted (nil)
}
```

### Update Structs

**Purpose:** Modify an existing resource

**Pattern:** All fields optional (pointers or slices with omitempty)

```go
type ResourceUpdate struct {
    // All fields optional (pointers)
    URI         *string  `json:"uri,omitempty"`
    Name        *string  `json:"name,omitempty"`
    Description *string  `json:"description,omitempty"`
    MimeType    *string  `json:"mimeType,omitempty"`
    Template    *string  `json:"template,omitempty"`
    Content     any      `json:"content,omitempty"`

    // All arrays optional (direct slices)
    Tags []string `json:"tags,omitempty"`
}
```

**Usage:**
```go
// Update only name
update := &contextforge.ResourceUpdate{
    Name: contextforge.String("new-name"),
}

// Update name and clear description
update := &contextforge.ResourceUpdate{
    Name:        contextforge.String("new-name"),
    Description: contextforge.String(""),
}

// Update tags (replace with new set)
update := &contextforge.ResourceUpdate{
    Tags: []string{"new-tag1", "new-tag2"},
}

// Clear all tags
update := &contextforge.ResourceUpdate{
    Tags: []string{},
}
```

## Code Examples

### Example 1: Partial Update (Update Only Changed Fields)

```go
// User wants to update only the description, leaving name and tags unchanged
update := &contextforge.ResourceUpdate{
    Description: contextforge.String("Updated description"),
    // Name is nil - won't be sent
    // Tags is nil - won't be sent
}

_, _, err := client.Resources.Update(ctx, resourceID, update)
```

**API receives:**
```json
{
  "description": "Updated description"
}
```

**Result:** Only description is updated, name and tags remain unchanged.

### Example 2: Clear a Field

```go
// User wants to clear the description
update := &contextforge.ResourceUpdate{
    Description: contextforge.String(""),
}

_, _, err := client.Resources.Update(ctx, resourceID, update)
```

**API receives:**
```json
{
  "description": ""
}
```

**Result:** Description is set to empty string.

### Example 3: Clear All Tags

```go
// User wants to remove all tags
update := &contextforge.ResourceUpdate{
    Tags: []string{},
}

_, _, err := client.Resources.Update(ctx, resourceID, update)
```

**API receives:**
```json
{
  "tags": []
}
```

**Result:** All tags are removed from the resource.

### Example 4: Set New Tags (Replace Existing)

```go
// User wants to replace tags with a new set
update := &contextforge.ResourceUpdate{
    Tags: []string{"production", "critical"},
}

_, _, err := client.Resources.Update(ctx, resourceID, update)
```

**API receives:**
```json
{
  "tags": ["production", "critical"]
}
```

**Result:** Tags are replaced with the new set.

### Example 5: Don't Touch Tags

```go
// User wants to update description but not modify tags
update := &contextforge.ResourceUpdate{
    Description: contextforge.String("New description"),
    Tags:        nil, // Explicitly nil
}

_, _, err := client.Resources.Update(ctx, resourceID, update)
```

**API receives:**
```json
{
  "description": "New description"
}
```

**Result:** Description updated, tags remain unchanged.

### Example 6: Update Multiple Fields

```go
// User wants to update name, description, and tags
update := &contextforge.ResourceUpdate{
    Name:        contextforge.String("new-name"),
    Description: contextforge.String("New description"),
    Tags:        []string{"updated", "tags"},
}

_, _, err := client.Resources.Update(ctx, resourceID, update)
```

**API receives:**
```json
{
  "name": "new-name",
  "description": "New description",
  "tags": ["updated", "tags"]
}
```

## Pointer Helper Functions

The SDK provides helper functions in `contextforge/pointers.go` for creating and dereferencing pointers.

### Creating Pointers

```go
// String pointers
name := contextforge.String("my-resource")
emptyStr := contextforge.String("")

// Integer pointers
count := contextforge.Int(10)
zero := contextforge.Int(0)
limit := contextforge.Int64(1000)

// Boolean pointers
enabled := contextforge.Bool(true)
disabled := contextforge.Bool(false)

// Float pointers
weight := contextforge.Float64(0.95)

// Time pointers
now := contextforge.Time(time.Now())
```

### Dereferencing Pointers (with Zero-Value Fallback)

```go
// String values
name := contextforge.StringValue(resource.Name)      // "my-resource"
empty := contextforge.StringValue(nil)               // "" (empty string)

// Integer values
count := contextforge.IntValue(resource.Count)       // 10
zero := contextforge.IntValue(nil)                   // 0

// Boolean values
enabled := contextforge.BoolValue(resource.Enabled)  // true
fallback := contextforge.BoolValue(nil)              // false

// Time values
timestamp := contextforge.TimeValue(resource.CreatedAt) // time.Time
zeroTime := contextforge.TimeValue(nil)                 // time.Time{} (zero time)
```

### Usage in Application Code

```go
// Building an update request
update := &contextforge.ResourceUpdate{
    Name:        contextforge.String("updated-name"),
    Description: contextforge.String(""),  // Clear description
}

// Processing a response
if resource.Description != nil {
    fmt.Printf("Description: %s\n", *resource.Description)
}
// OR using helper
desc := contextforge.StringValue(resource.Description)
fmt.Printf("Description: %s\n", desc) // Empty string if nil
```

## Industry Precedent

This three-state pattern is the standard approach in major Go SDKs:

### google/go-github

```go
// From github.com/google/go-github/github/repos.go
type Repository struct {
    Name        *string   `json:"name,omitempty"`
    Description *string   `json:"description,omitempty"`
    Topics      []string  `json:"topics,omitempty"`
}

// Helper functions
repo := &github.Repository{
    Name:        github.String("my-repo"),
    Description: github.String(""),  // Clear description
    Topics:      []string{},         // Clear topics
}
```

### hashicorp/go-tfe

```go
// From github.com/hashicorp/go-tfe/workspace.go
type WorkspaceUpdateOptions struct {
    Name        *string `jsonapi:"attr,name,omitempty"`
    AutoApply   *bool   `jsonapi:"attr,auto-apply,omitempty"`
    Description *string `jsonapi:"attr,description,omitempty"`
}

// Usage
opts := &tfe.WorkspaceUpdateOptions{
    Name:        tfe.String("new-name"),
    Description: tfe.String(""),  // Clear description
}
```

### AWS SDK Go v2

```go
// From github.com/aws/aws-sdk-go-v2/service/s3
type PutBucketTaggingInput struct {
    Bucket  *string
    Tagging *Tagging
}

// Usage
input := &s3.PutBucketTaggingInput{
    Bucket: aws.String("my-bucket"),
    Tagging: &types.Tagging{
        TagSet: []types.Tag{}, // Clear all tags
    },
}
```

## Common Pitfalls

### ❌ Pitfall 1: Using Pointer to Slice

```go
// DON'T DO THIS
type ResourceUpdate struct {
    Tags *[]string `json:"tags,omitempty"`  // Incorrect!
}

// This complicates usage
tags := []string{"tag1"}
update := &ResourceUpdate{
    Tags: &tags,  // Extra indirection
}
```

**Solution:** Use direct slice
```go
type ResourceUpdate struct {
    Tags []string `json:"tags,omitempty"`  // Correct
}

update := &ResourceUpdate{
    Tags: []string{"tag1"},  // Simpler
}
```

### ❌ Pitfall 2: Forgetting omitempty

```go
// DON'T DO THIS
type ResourceUpdate struct {
    Name *string `json:"name"`  // Missing omitempty!
}

// nil fields will be sent as null
update := &ResourceUpdate{}
// JSON: {"name": null}  - API may reject this
```

**Solution:** Always use omitempty
```go
type ResourceUpdate struct {
    Name *string `json:"name,omitempty"`  // Correct
}

update := &ResourceUpdate{}
// JSON: {}  - nil fields omitted
```

### ❌ Pitfall 3: Sending Unintended Zero Values

```go
// DON'T DO THIS
update := &ResourceUpdate{
    Name:        contextforge.String("new-name"),
    Description: contextforge.String(""),  // Accidentally clears!
}
```

**Solution:** Only set fields you intend to update
```go
update := &ResourceUpdate{
    Name: contextforge.String("new-name"),
    // Description is nil - won't be updated
}
```

### ❌ Pitfall 4: Confusion Between nil and Empty Slice

```go
// These are DIFFERENT
update1 := &ResourceUpdate{
    Tags: nil,        // Don't update tags
}

update2 := &ResourceUpdate{
    Tags: []string{}, // Clear all tags
}
```

**Solution:** Be explicit about intent
```go
// Don't touch tags
update := &ResourceUpdate{
    Name: contextforge.String("new-name"),
    Tags: nil,  // Explicitly nil
}

// Clear tags
update := &ResourceUpdate{
    Name: contextforge.String("new-name"),
    Tags: []string{},  // Explicitly empty
}
```

### ❌ Pitfall 5: Using == nil for Slice Checks

```go
// DON'T DO THIS (ambiguous)
if update.Tags == nil {
    // Is this "don't update" or "clear tags"?
}
```

**Solution:** Check length for intent
```go
// Check if tags should be updated
if update.Tags != nil {
    if len(update.Tags) == 0 {
        // Clear tags (empty slice sent to API)
    } else {
        // Set tags (populated slice sent to API)
    }
} else {
    // Don't update tags (omitted from API request)
}
```

## Testing Considerations

### Unit Testing with Mock Server

```go
func TestResourceUpdate_PartialUpdate(t *testing.T) {
    // Create mock server that validates request
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req contextforge.ResourceUpdate
        json.NewDecoder(r.Body).Decode(&req)

        // Verify only name was sent
        if req.Name == nil {
            t.Error("Expected name to be set")
        }
        if req.Description != nil {
            t.Error("Expected description to be nil")
        }
        if req.Tags != nil {
            t.Error("Expected tags to be nil")
        }

        // Return mock response
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(&contextforge.Resource{
            Name: *req.Name,
        })
    }))
    defer server.Close()

    client, _ := contextforge.NewClient(nil, server.URL, "token")

    // Test partial update
    update := &contextforge.ResourceUpdate{
        Name: contextforge.String("new-name"),
    }

    _, _, err := client.Resources.Update(context.Background(), "resource-id", update)
    if err != nil {
        t.Fatalf("Update failed: %v", err)
    }
}
```

### Testing Clear vs Omit

```go
func TestResourceUpdate_ClearTags(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req contextforge.ResourceUpdate
        json.NewDecoder(r.Body).Decode(&req)

        // Verify tags were sent as empty array
        if req.Tags == nil {
            t.Error("Expected tags to be non-nil (should be empty slice)")
        }
        if len(req.Tags) != 0 {
            t.Errorf("Expected tags to be empty, got %v", req.Tags)
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(&contextforge.Resource{})
    }))
    defer server.Close()

    client, _ := contextforge.NewClient(nil, server.URL, "token")

    // Test clearing tags
    update := &contextforge.ResourceUpdate{
        Tags: []string{},  // Empty slice, not nil
    }

    _, _, err := client.Resources.Update(context.Background(), "resource-id", update)
    if err != nil {
        t.Fatalf("Update failed: %v", err)
    }
}
```

## Quick Reference

| Intent | Scalar Fields | Array Fields |
|--------|---------------|--------------|
| Don't update | `nil` | `nil` |
| Clear/Empty | `contextforge.String("")` | `[]string{}` |
| Set value | `contextforge.String("value")` | `[]string{"value"}` |

| Struct Type | Required Fields | Optional Fields |
|-------------|-----------------|-----------------|
| Create | Direct values | Pointers/slices |
| Update | N/A (all optional) | Pointers/slices |

| Helper Function | Purpose | Example |
|-----------------|---------|---------|
| `String(v)` | Create string pointer | `contextforge.String("value")` |
| `Int(v)` | Create int pointer | `contextforge.Int(10)` |
| `Bool(v)` | Create bool pointer | `contextforge.Bool(true)` |
| `StringValue(v)` | Dereference with fallback | `contextforge.StringValue(ptr)` |
| `IntValue(v)` | Dereference with fallback | `contextforge.IntValue(ptr)` |
| `BoolValue(v)` | Dereference with fallback | `contextforge.BoolValue(ptr)` |

## Further Reading

- [google/go-github pointer pattern rationale](https://github.com/google/go-github/issues/19)
- [Go REST APIs and Pointers (willnorris.com)](https://willnorris.com/2014/05/go-rest-apis-and-pointers)
- [AWS SDK Go v2 Optional Fields](https://github.com/aws/aws-sdk-go-v2/issues/2162)
- [Protocol Buffers Go Tutorial](https://protobuf.dev/getting-started/gotutorial/) (repeated fields pattern)
