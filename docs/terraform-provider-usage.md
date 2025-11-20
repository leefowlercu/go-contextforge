# Terraform Provider Usage Guide

## Overview

This guide demonstrates how to build a Terraform provider for ContextForge using the [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework) and the go-contextforge SDK.

The key challenge in Terraform provider development is mapping Terraform's declarative state model to the SDK's three-state system. This guide shows how to correctly implement Create, Read, Update, and Delete (CRUD) operations to leverage the SDK's partial update capabilities.

### Why This Matters

Terraform manages **desired state** - users declare what they want, and Terraform makes it happen. The go-contextforge SDK uses a **three-state system** for updates (nil = don't send, empty = clear, value = set). Correctly bridging these two models enables:

- **Partial updates** - only changed fields sent to API
- **Tag management** - clearing vs not updating tags
- **`ignore_changes` support** - Terraform can ignore specific fields
- **Minimal API calls** - efficient resource management

## Terraform's State Model

The Terraform Plugin Framework represents configuration values using three states:

### 1. Null (Not Configured)

**Meaning:** User didn't configure this attribute in their `.tf` file

**Framework Type:** `types.StringNull()`, `types.ListNull()`, etc.

**Example HCL:**
```hcl
resource "contextforge_resource" "example" {
  name = "my-resource"
  uri  = "file:///config.json"
  # description not configured - Null
  # tags not configured - Null
}
```

### 2. Unknown (Known After Apply)

**Meaning:** Value will be determined during apply (e.g., computed fields)

**Framework Type:** `types.StringUnknown()`, `types.ListUnknown()`, etc.

**Example:** Computed fields like `id`, `created_at`

### 3. Known (Has Value)

**Meaning:** User configured a specific value (may be zero value like `""` or `[]`)

**Framework Type:** `types.StringValue("value")`, `types.ListValueMust(...)`, etc.

**Example HCL:**
```hcl
resource "contextforge_resource" "example" {
  name        = "my-resource"                    # Known: "my-resource"
  description = ""                               # Known: "" (empty string)
  tags        = []                               # Known: [] (empty list)
}
```

## Framework to SDK Type Mapping

### Basic Mapping

| Framework Type | Go Type | SDK Type | Usage |
|----------------|---------|----------|-------|
| `types.String` | `string` | `*string` | `contextforge.String(val.ValueString())` |
| `types.Int64` | `int64` | `*int64` | `contextforge.Int64(val.ValueInt64())` |
| `types.Bool` | `bool` | `*bool` | `contextforge.Bool(val.ValueBool())` |
| `types.List` | `[]T` | `[]T` | Extract elements as Go slice |
| `types.Map` | `map[K]V` | `map[K]V` | Extract elements as Go map |

### Null Handling

The key to correct implementation is checking `.IsNull()` before converting:

```go
var description *string
if !plan.Description.IsNull() {
    description = contextforge.String(plan.Description.ValueString())
}
// If IsNull() == true, description stays nil
```

## CRUD Operations

### Create Operation

**Goal:** Create a new resource with all user-configured fields

**Pattern:** Map all Known fields from plan to SDK create struct

```go
func (r *resourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // 1. Read plan
    var plan resourceResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Build SDK create struct
    createReq := &contextforge.ResourceCreate{
        // Required fields (always present in plan)
        URI:     plan.URI.ValueString(),
        Name:    plan.Name.ValueString(),
        Content: plan.Content.ValueString(),
    }

    // Optional scalar fields - only set if configured
    if !plan.Description.IsNull() {
        createReq.Description = contextforge.String(plan.Description.ValueString())
    }
    if !plan.MimeType.IsNull() {
        createReq.MimeType = contextforge.String(plan.MimeType.ValueString())
    }

    // Optional list fields - only set if configured
    if !plan.Tags.IsNull() {
        var tags []string
        resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
        if resp.Diagnostics.HasError() {
            return
        }
        createReq.Tags = tags  // Can be empty slice if user set tags = []
    }

    // Optional create options (team, visibility)
    var opts *contextforge.ResourceCreateOptions
    if !plan.TeamID.IsNull() || !plan.Visibility.IsNull() {
        opts = &contextforge.ResourceCreateOptions{}
        if !plan.TeamID.IsNull() {
            opts.TeamID = contextforge.String(plan.TeamID.ValueString())
        }
        if !plan.Visibility.IsNull() {
            opts.Visibility = contextforge.String(plan.Visibility.ValueString())
        }
    }

    // 3. Call SDK
    created, _, err := r.client.Resources.Create(ctx, createReq, opts)
    if err != nil {
        resp.Diagnostics.AddError("Error creating resource", err.Error())
        return
    }

    // 4. Map response to state
    plan.ID = types.StringValue(created.ID.String())
    plan.IsActive = types.BoolValue(created.IsActive)
    if created.CreatedAt != nil {
        plan.CreatedAt = types.StringValue(created.CreatedAt.Format(time.RFC3339))
    }

    // Set state
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### Read Operation

**Goal:** Populate Terraform state from API response

**Pattern:** Map all API fields to Framework types using pointer helpers

```go
func (r *resourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    // 1. Read current state (to get ID)
    var state resourceResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Call SDK
    resource, _, err := r.client.Resources.Get(ctx, state.ID.ValueString())
    if err != nil {
        // Handle 404 by removing from state
        if strings.Contains(err.Error(), "404") {
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.AddError("Error reading resource", err.Error())
        return
    }

    // 3. Map response to state
    state.ID = types.StringValue(resource.ID.String())
    state.URI = types.StringValue(resource.URI)
    state.Name = types.StringValue(resource.Name)
    state.IsActive = types.BoolValue(resource.IsActive)

    // Handle optional pointer fields
    if resource.Description != nil {
        state.Description = types.StringValue(*resource.Description)
    } else {
        state.Description = types.StringNull()
    }

    if resource.MimeType != nil {
        state.MimeType = types.StringValue(*resource.MimeType)
    } else {
        state.MimeType = types.StringNull()
    }

    // Handle optional slice fields
    if resource.Tags != nil && len(resource.Tags) > 0 {
        tags, diags := types.ListValueFrom(ctx, types.StringType, resource.Tags)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
            return
        }
        state.Tags = tags
    } else {
        state.Tags = types.ListNull(types.StringType)
    }

    // Timestamps
    if resource.CreatedAt != nil {
        state.CreatedAt = types.StringValue(resource.CreatedAt.Format(time.RFC3339))
    }
    if resource.UpdatedAt != nil {
        state.UpdatedAt = types.StringValue(resource.UpdatedAt.Format(time.RFC3339))
    }

    // 4. Set state
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
```

### Update Operation

**Goal:** Send only changed fields to API using SDK's three-state system

**Pattern:** Compare plan vs state, set only changed fields to non-nil

```go
func (r *resourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    // 1. Read plan and state
    var plan, state resourceResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Build SDK update struct (all fields start as nil)
    updateReq := &contextforge.ResourceUpdate{}
    hasChanges := false

    // 3. Compare plan vs state for each field

    // URI changed?
    if !plan.URI.Equal(state.URI) {
        updateReq.URI = contextforge.String(plan.URI.ValueString())
        hasChanges = true
    }

    // Name changed?
    if !plan.Name.Equal(state.Name) {
        updateReq.Name = contextforge.String(plan.Name.ValueString())
        hasChanges = true
    }

    // Description changed?
    if !plan.Description.Equal(state.Description) {
        if plan.Description.IsNull() {
            // User removed description from config - DON'T UPDATE
            // Leave updateReq.Description as nil
        } else {
            // User set description (could be empty string to clear)
            updateReq.Description = contextforge.String(plan.Description.ValueString())
            hasChanges = true
        }
    }

    // MimeType changed?
    if !plan.MimeType.Equal(state.MimeType) {
        if plan.MimeType.IsNull() {
            // User removed mime_type from config - DON'T UPDATE
        } else {
            updateReq.MimeType = contextforge.String(plan.MimeType.ValueString())
            hasChanges = true
        }
    }

    // Tags changed?
    if !plan.Tags.Equal(state.Tags) {
        if plan.Tags.IsNull() {
            // User removed tags from config - DON'T UPDATE
            // Leave updateReq.Tags as nil
        } else {
            // User set tags (could be empty list to clear)
            var tags []string
            resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
            if resp.Diagnostics.HasError() {
                return
            }
            updateReq.Tags = tags  // Will be [] if user set tags = []
            hasChanges = true
        }
    }

    // 4. Only call API if something changed
    if !hasChanges {
        // No changes detected, just refresh state
        resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
        return
    }

    // 5. Call SDK
    updated, _, err := r.client.Resources.Update(ctx, plan.ID.ValueString(), updateReq)
    if err != nil {
        resp.Diagnostics.AddError("Error updating resource", err.Error())
        return
    }

    // 6. Update state from response
    if updated.Description != nil {
        plan.Description = types.StringValue(*updated.Description)
    }
    if updated.Tags != nil {
        tags, diags := types.ListValueFrom(ctx, types.StringType, updated.Tags)
        resp.Diagnostics.Append(diags...)
        if !resp.Diagnostics.HasError() {
            plan.Tags = tags
        }
    }
    if updated.UpdatedAt != nil {
        plan.UpdatedAt = types.StringValue(updated.UpdatedAt.Format(time.RFC3339))
    }

    // 7. Set state
    resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

### Delete Operation

**Goal:** Remove resource from API and Terraform state

**Pattern:** Simple API call, no complex mapping needed

```go
func (r *resourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    // 1. Read current state (to get ID)
    var state resourceResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // 2. Call SDK
    _, err := r.client.Resources.Delete(ctx, state.ID.ValueString())
    if err != nil {
        // Ignore 404 errors (resource already deleted)
        if !strings.Contains(err.Error(), "404") {
            resp.Diagnostics.AddError("Error deleting resource", err.Error())
            return
        }
    }

    // 3. State automatically removed by Framework
}
```

## Tags/Arrays Handling

Tags require special attention to correctly implement the three-state system.

### Scenario 1: User Removes Tags from Config

**HCL before:**
```hcl
resource "contextforge_resource" "example" {
  name = "my-resource"
  tags = ["tag1", "tag2"]
}
```

**HCL after:**
```hcl
resource "contextforge_resource" "example" {
  name = "my-resource"
  # tags removed
}
```

**Provider behavior:**
```go
// In Update()
if !plan.Tags.Equal(state.Tags) {
    if plan.Tags.IsNull() {
        // Tags removed from config - DON'T UPDATE
        // Leave updateReq.Tags as nil
        // Existing tags in API remain unchanged
    }
}
```

**Result:** Existing tags remain unchanged in API

### Scenario 2: User Sets Tags to Empty List

**HCL:**
```hcl
resource "contextforge_resource" "example" {
  name = "my-resource"
  tags = []  # Explicitly empty
}
```

**Provider behavior:**
```go
// In Update()
if !plan.Tags.Equal(state.Tags) {
    if !plan.Tags.IsNull() {
        var tags []string
        plan.Tags.ElementsAs(ctx, &tags, false)
        updateReq.Tags = tags  // tags = []string{} (empty slice)
    }
}
```

**SDK behavior:**
```go
// In SDK, omitempty applies
[]string{} is NOT considered empty, so it's included in JSON
// JSON: {"tags": []}
```

**Result:** All tags are cleared from the resource

### Scenario 3: User Sets New Tags

**HCL:**
```hcl
resource "contextforge_resource" "example" {
  name = "my-resource"
  tags = ["production", "critical"]
}
```

**Provider behavior:**
```go
// In Update()
if !plan.Tags.Equal(state.Tags) {
    var tags []string
    plan.Tags.ElementsAs(ctx, &tags, false)
    updateReq.Tags = tags  // tags = []string{"production", "critical"}
}
```

**Result:** Tags replaced with new values

### Scenario 4: User Doesn't Modify Tags

**HCL (unchanged):**
```hcl
resource "contextforge_resource" "example" {
  name = "my-resource"
  tags = ["tag1"]
}
```

**Provider behavior:**
```go
// In Update()
if plan.Tags.Equal(state.Tags) {
    // Tags haven't changed - don't include in update
    // updateReq.Tags remains nil
}
```

**Result:** Tags remain unchanged in API

### Summary Table

| User Action | plan.Tags | state.Tags | Equal? | updateReq.Tags | API Request | Result |
|-------------|-----------|------------|--------|----------------|-------------|---------|
| Remove tags | Null | Known | ❌ No | `nil` | Omitted | Unchanged |
| Set to `[]` | Known `[]` | Known | ❌ No | `[]string{}` | `"tags": []` | Cleared |
| Set to `["a"]` | Known `["a"]` | Known | ❌ No | `[]string{"a"}` | `"tags": ["a"]` | Replaced |
| No change | Known `["a"]` | Known `["a"]` | ✅ Yes | `nil` | Omitted | Unchanged |

## Complete Resource Implementation

### Resource Schema

```go
func (r *resourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages a ContextForge resource",
        Attributes: map[string]schema.Attribute{
            // Computed fields
            "id": schema.StringAttribute{
                Description: "Resource ID",
                Computed:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "is_active": schema.BoolAttribute{
                Description: "Whether the resource is active",
                Computed:    true,
            },
            "created_at": schema.StringAttribute{
                Description: "Resource creation timestamp (RFC3339)",
                Computed:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "updated_at": schema.StringAttribute{
                Description: "Resource last update timestamp (RFC3339)",
                Computed:    true,
            },

            // Required fields
            "uri": schema.StringAttribute{
                Description: "Resource URI (e.g., file:///path/to/file)",
                Required:    true,
            },
            "name": schema.StringAttribute{
                Description: "Resource name",
                Required:    true,
            },
            "content": schema.StringAttribute{
                Description: "Resource content",
                Required:    true,
            },

            // Optional fields
            "description": schema.StringAttribute{
                Description: "Resource description",
                Optional:    true,
            },
            "mime_type": schema.StringAttribute{
                Description: "MIME type (e.g., application/json)",
                Optional:    true,
            },
            "tags": schema.ListAttribute{
                Description: "Resource tags",
                Optional:    true,
                ElementType: types.StringType,
            },
            "team_id": schema.StringAttribute{
                Description: "Team ID for team-scoped resources",
                Optional:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),
                },
            },
            "visibility": schema.StringAttribute{
                Description: "Resource visibility (public, private, etc.)",
                Optional:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),
                },
            },
        },
    }
}
```

### Resource Model

```go
type resourceResourceModel struct {
    // Computed fields
    ID        types.String `tfsdk:"id"`
    IsActive  types.Bool   `tfsdk:"is_active"`
    CreatedAt types.String `tfsdk:"created_at"`
    UpdatedAt types.String `tfsdk:"updated_at"`

    // Required fields
    URI     types.String `tfsdk:"uri"`
    Name    types.String `tfsdk:"name"`
    Content types.String `tfsdk:"content"`

    // Optional fields
    Description types.String `tfsdk:"description"`
    MimeType    types.String `tfsdk:"mime_type"`
    Tags        types.List   `tfsdk:"tags"`
    TeamID      types.String `tfsdk:"team_id"`
    Visibility  types.String `tfsdk:"visibility"`
}
```

### Helper: Map Create Request

```go
func (r *resourceResource) buildCreateRequest(ctx context.Context, plan resourceResourceModel, diags *diag.Diagnostics) (*contextforge.ResourceCreate, *contextforge.ResourceCreateOptions) {
    // Build create request
    req := &contextforge.ResourceCreate{
        URI:     plan.URI.ValueString(),
        Name:    plan.Name.ValueString(),
        Content: plan.Content.ValueString(),
    }

    // Optional fields
    if !plan.Description.IsNull() {
        req.Description = contextforge.String(plan.Description.ValueString())
    }
    if !plan.MimeType.IsNull() {
        req.MimeType = contextforge.String(plan.MimeType.ValueString())
    }
    if !plan.Tags.IsNull() {
        var tags []string
        diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
        if !diags.HasError() {
            req.Tags = tags
        }
    }

    // Build options
    var opts *contextforge.ResourceCreateOptions
    if !plan.TeamID.IsNull() || !plan.Visibility.IsNull() {
        opts = &contextforge.ResourceCreateOptions{}
        if !plan.TeamID.IsNull() {
            opts.TeamID = contextforge.String(plan.TeamID.ValueString())
        }
        if !plan.Visibility.IsNull() {
            opts.Visibility = contextforge.String(plan.Visibility.ValueString())
        }
    }

    return req, opts
}
```

### Helper: Map Update Request

```go
func (r *resourceResource) buildUpdateRequest(ctx context.Context, plan, state resourceResourceModel, diags *diag.Diagnostics) (*contextforge.ResourceUpdate, bool) {
    req := &contextforge.ResourceUpdate{}
    hasChanges := false

    // Check each field for changes
    if !plan.URI.Equal(state.URI) {
        req.URI = contextforge.String(plan.URI.ValueString())
        hasChanges = true
    }

    if !plan.Name.Equal(state.Name) {
        req.Name = contextforge.String(plan.Name.ValueString())
        hasChanges = true
    }

    if !plan.Description.Equal(state.Description) {
        if !plan.Description.IsNull() {
            req.Description = contextforge.String(plan.Description.ValueString())
            hasChanges = true
        }
    }

    if !plan.MimeType.Equal(state.MimeType) {
        if !plan.MimeType.IsNull() {
            req.MimeType = contextforge.String(plan.MimeType.ValueString())
            hasChanges = true
        }
    }

    if !plan.Tags.Equal(state.Tags) {
        if !plan.Tags.IsNull() {
            var tags []string
            diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
            if !diags.HasError() {
                req.Tags = tags
                hasChanges = true
            }
        }
    }

    return req, hasChanges
}
```

### Helper: Map Response to State

```go
func (r *resourceResource) mapResponseToState(ctx context.Context, resource *contextforge.Resource, state *resourceResourceModel, diags *diag.Diagnostics) {
    // Map all fields from API response to Terraform state
    state.ID = types.StringValue(resource.ID.String())
    state.URI = types.StringValue(resource.URI)
    state.Name = types.StringValue(resource.Name)
    state.IsActive = types.BoolValue(resource.IsActive)

    // Optional pointer fields
    if resource.Description != nil {
        state.Description = types.StringValue(*resource.Description)
    } else {
        state.Description = types.StringNull()
    }

    if resource.MimeType != nil {
        state.MimeType = types.StringValue(*resource.MimeType)
    } else {
        state.MimeType = types.StringNull()
    }

    // Optional slice fields
    if resource.Tags != nil && len(resource.Tags) > 0 {
        tags, d := types.ListValueFrom(ctx, types.StringType, resource.Tags)
        diags.Append(d...)
        if !diags.HasError() {
            state.Tags = tags
        }
    } else {
        state.Tags = types.ListNull(types.StringType)
    }

    // Timestamps
    if resource.CreatedAt != nil {
        state.CreatedAt = types.StringValue(resource.CreatedAt.Format(time.RFC3339))
    }
    if resource.UpdatedAt != nil {
        state.UpdatedAt = types.StringValue(resource.UpdatedAt.Format(time.RFC3339))
    }
}
```

## Testing Strategy

### Unit Testing Update Logic

```go
func TestResourceUpdate_PartialUpdate(t *testing.T) {
    // Setup
    ctx := context.Background()
    plan := resourceResourceModel{
        ID:          types.StringValue("resource-1"),
        Name:        types.StringValue("new-name"),
        Description: types.StringNull(),  // Removed from config
        Tags:        types.ListNull(types.StringType),  // Removed from config
    }
    state := resourceResourceModel{
        ID:          types.StringValue("resource-1"),
        Name:        types.StringValue("old-name"),
        Description: types.StringValue("old-description"),
        Tags:        types.ListValueMust(types.StringType, []attr.Value{
            types.StringValue("tag1"),
        }),
    }

    // Build update request
    var diags diag.Diagnostics
    updateReq, hasChanges := buildUpdateRequest(ctx, plan, state, &diags)

    // Assertions
    assert.True(t, hasChanges)
    assert.NotNil(t, updateReq.Name)
    assert.Equal(t, "new-name", *updateReq.Name)
    assert.Nil(t, updateReq.Description)  // Removed, should be nil
    assert.Nil(t, updateReq.Tags)         // Removed, should be nil
}
```

### Acceptance Testing (Real API)

```go
func TestAccResource_ClearTags(t *testing.T) {
    resource.Test(t, resource.TestCase{
        ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            {
                // Create with tags
                Config: `
resource "contextforge_resource" "test" {
  name    = "test-resource"
  uri     = "file:///test.json"
  content = "{}"
  tags    = ["tag1", "tag2"]
}
`,
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("contextforge_resource.test", "tags.#", "2"),
                ),
            },
            {
                // Clear tags
                Config: `
resource "contextforge_resource" "test" {
  name    = "test-resource"
  uri     = "file:///test.json"
  content = "{}"
  tags    = []  # Explicitly empty
}
`,
                Check: resource.ComposeTestCheckFunc(
                    resource.TestCheckResourceAttr("contextforge_resource.test", "tags.#", "0"),
                ),
            },
        },
    })
}
```

## Best Practices

### 1. Always Check IsNull() Before Converting

```go
// ✅ CORRECT
if !plan.Description.IsNull() {
    createReq.Description = contextforge.String(plan.Description.ValueString())
}

// ❌ INCORRECT - will panic if null
createReq.Description = contextforge.String(plan.Description.ValueString())
```

### 2. Use Equal() for Change Detection

```go
// ✅ CORRECT
if !plan.Name.Equal(state.Name) {
    updateReq.Name = contextforge.String(plan.Name.ValueString())
}

// ❌ INCORRECT - doesn't handle all Framework types correctly
if plan.Name.ValueString() != state.Name.ValueString() {
    updateReq.Name = contextforge.String(plan.Name.ValueString())
}
```

### 3. Handle Null vs Empty for Lists

```go
// ✅ CORRECT - distinguishes null from empty
if !plan.Tags.Equal(state.Tags) {
    if plan.Tags.IsNull() {
        // User removed tags - don't update
    } else {
        var tags []string
        plan.Tags.ElementsAs(ctx, &tags, false)
        updateReq.Tags = tags  // Will be [] if empty
    }
}

// ❌ INCORRECT - can't distinguish null from empty
var tags []string
plan.Tags.ElementsAs(ctx, &tags, false)
updateReq.Tags = tags  // Always set, even if null
```

### 4. Use Plan Modifiers for Computed Fields

```go
"id": schema.StringAttribute{
    Computed: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.UseStateForUnknown(),  // Preserve ID during updates
    },
},
```

### 5. Mark ForceNew Fields Appropriately

```go
"team_id": schema.StringAttribute{
    Optional: true,
    PlanModifiers: []planmodifier.String{
        stringplanmodifier.RequiresReplace(),  // Can't change team after creation
    },
},
```

### 6. Handle SDK Errors Gracefully

```go
_, err := r.client.Resources.Update(ctx, id, updateReq)
if err != nil {
    resp.Diagnostics.AddError(
        "Error Updating Resource",
        fmt.Sprintf("Could not update resource %s: %s", id, err.Error()),
    )
    return
}
```

### 7. Support Import Operations

```go
func (r *resourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    // Import by ID
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

## Common Patterns

### Pattern: Optional Fields with Defaults

If the API has defaults but you want users to be able to clear them:

```go
"visibility": schema.StringAttribute{
    Optional:    true,
    Computed:    true,  // API provides default
    Default:     stringdefault.StaticString("public"),
    Description: "Resource visibility (default: public)",
},
```

In Update:
```go
if !plan.Visibility.Equal(state.Visibility) {
    if !plan.Visibility.IsNull() {
        updateReq.Visibility = contextforge.String(plan.Visibility.ValueString())
    } else {
        // User removed visibility - let API use default
        // Don't set updateReq.Visibility
    }
}
```

### Pattern: Nested Blocks

For complex nested structures:

```go
// Schema
"server": schema.SingleNestedAttribute{
    Optional: true,
    Attributes: map[string]schema.Attribute{
        "name": schema.StringAttribute{Required: true},
        "port": schema.Int64Attribute{Required: true},
    },
},

// Model
type resourceModel struct {
    Server types.Object `tfsdk:"server"`
}

type serverModel struct {
    Name types.String `tfsdk:"name"`
    Port types.Int64  `tfsdk:"port"`
}

// In Update
if !plan.Server.Equal(state.Server) {
    if !plan.Server.IsNull() {
        var server serverModel
        diags.Append(plan.Server.As(ctx, &server, basetypes.ObjectAsOptions{})...)
        // Build SDK request from server model
    }
}
```

## Further Reading

- [Terraform Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [Handling Data - Accessing Values](https://developer.hashicorp.com/terraform/plugin/framework/handling-data/accessing-values)
- [Resources - Update](https://developer.hashicorp.com/terraform/plugin/framework/resources/update)
- [go-contextforge Three-State System](./three-state-system.md)
- [terraform-provider-github](https://github.com/integrations/terraform-provider-github) - Example using google/go-github SDK
- [terraform-provider-aws](https://github.com/hashicorp/terraform-provider-aws) - Example using AWS SDK
