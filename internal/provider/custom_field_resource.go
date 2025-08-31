// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/i10416/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CustomFieldResource{}
var _ resource.ResourceWithImportState = &CustomFieldResource{}

func newCustomFieldResource() resource.Resource {
	return &CustomFieldResource{}
}

type CustomFieldResource struct {
	client *sendgrid.Client
}

type CustomFieldResourceModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (r *CustomFieldResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_field"
}

func (r *CustomFieldResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Twilio SendGrid's CustomFields feature allows you to receive notifications regarding your usage or program statistics from SendGrid at an email address you specify.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The ID of CustomField",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of a CustomField. Example: foo",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of CustomField you want to create. Can be either usage_limit or stats_notification. Example: usage_limit",
				Required:            true,
				Validators: []validator.String{
					stringOneOf(
						"text",
						"number",
						"date",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *CustomFieldResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*sendgrid.Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *sendgrid.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *CustomFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CustomFieldResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := validateCustomField(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating CustomField",
			err.Error(),
		)
		return
	}

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateCustomField(ctx, &sendgrid.InputCreateCustomField{
			Name: plan.Name.ValueString(),
			Type: plan.Type.ValueString(),
		})
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating CustomField",
			fmt.Sprintf("Unable to create CustomField, got error: %s", err),
		)
		return
	}

	o, ok := res.(*sendgrid.CustomField)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating CustomField",
			"Failed to assert type *sendgrid.OutputCreateCustomField",
		)
		return
	}

	plan = CustomFieldResourceModel{
		ID:   types.Int64Value(o.ID),
		Name: types.StringValue(o.Name),
		Type: types.StringValue(o.Type),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *CustomFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CustomFieldResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueInt64()

	o, err := r.client.GetCustomField(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading CustomField",
			fmt.Sprintf("Unable to read CustomField (id: %d), got error: %e", id, err),
		)
		return
	}

	state.ID = types.Int64Value(id)
	state.Name = types.StringValue(o.Name)
	state.Type = types.StringValue(o.Type)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *CustomFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// var data, state CustomFieldResourceModel
	// resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	// resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// if resp.Diagnostics.HasError() {
	// 	return
	// }

	// id := state.ID.ValueString()
	// idInt64, err := strconv.ParseInt(id, 10, 64)
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Updating CustomField",
	// 		fmt.Sprintf("Unable to update CustomField, got error: %s", err),
	// 	)
	// 	return
	// }

	// o, err := r.client.UpdateCustomField(ctx, idInt64, &sendgrid.InputUpdateCustomField{
	// 	EmailTo:    data.Name.ValueString(),
	// 	Frequency:  data.Frequency.ValueString(),
	// 	Percentage: data.Percentage.ValueInt64(),
	// })
	// if err != nil {
	// 	resp.Diagnostics.AddError(
	// 		"Updating CustomField",
	// 		fmt.Sprintf("Unable to update CustomField, got error: %s", err),
	// 	)
	// 	return
	// }

	// data = CustomFieldResourceModel{
	// 	ID:   types.StringValue(strconv.FormatInt(o.ID, 10)),
	// 	Name: types.StringValue(o.Name),
	// 	Type: types.StringValue(o.Type),
	// }

	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	// if resp.Diagnostics.HasError() {
	// 	return
	// }
}

func (r *CustomFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CustomFieldResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_ = state.ID.ValueInt64()
}

func (r *CustomFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data CustomFieldResourceModel

	id := req.ID
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing CustomField",
			fmt.Sprintf("Unable to read CustomField, got error: %s", err),
		)
		return
	}

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	o, err := r.client.GetCustomField(ctx, idInt64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing CustomField",
			fmt.Sprintf("Unable to read CustomField, got error: %s", err),
		)
		return
	}

	data = CustomFieldResourceModel{
		ID:   types.Int64Value(idInt64),
		Name: types.StringValue(o.Name),
		Type: types.StringValue(o.Type),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func validateCustomField(_ *CustomFieldResourceModel) error {
	return nil
}
