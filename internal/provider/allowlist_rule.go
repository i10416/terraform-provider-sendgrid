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
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/i10416/sendgrid"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AllowlistRuleResource{}
var _ resource.ResourceWithImportState = &AllowlistRuleResource{}

func newAllowlistRuleResource() resource.Resource {
	return &AllowlistRuleResource{}
}

type AllowlistRuleResource struct {
	client *sendgrid.Client
}

type AllowlistRuleResourceModel struct {
	ID types.Int64  `tfsdk:"id"`
	Ip types.String `tfsdk:"ip"`
}

func (r *AllowlistRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_allowlist_rule"
}

func (r *AllowlistRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `Twilio SendGrid's AllowlistRules feature`,
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The ID of AllowlistRule",
				Computed:            true,
			},
			"ip": schema.StringAttribute{
				MarkdownDescription: "The ip to allow access. Example: 1.2.3.4",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *AllowlistRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AllowlistRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AllowlistRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := validateAllowlistRule(&plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating AllowlistRule",
			err.Error(),
		)
		return
	}

	// NOTE: Re-execute after the re-executable time has elapsed when a rate limit occurs
	res, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return r.client.CreateAllowlistRule(ctx, &sendgrid.InputCreateAllowlistRule{
			Ips: []sendgrid.InputCreateAllowlistRuleIp{
				{
					Ip: plan.Ip.ValueString(),
				},
			},
		})
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating AllowlistRule",
			fmt.Sprintf("Unable to create AllowlistRule, got error: %s", err),
		)
		return
	}

	o, ok := res.(sendgrid.OutputCreateAllowlistRule)
	if !ok {
		resp.Diagnostics.AddError(
			"Creating AllowlistRule",
			"Failed to assert type *sendgrid.OutputCreateAllowlistRule",
		)
		return
	}
	one := o.Result[0]

	plan = AllowlistRuleResourceModel{
		ID: types.Int64Value(one.ID),
		Ip: types.StringValue(one.Ip),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AllowlistRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AllowlistRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueInt64()

	o, err := r.client.GetAllowlistRule(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Reading AllowlistRule",
			fmt.Sprintf("Unable to read AllowlistRule (id: %d), got error: %e", id, err),
		)
		return
	}

	state.ID = types.Int64Value(id)
	state.Ip = types.StringValue(o.Ip)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AllowlistRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *AllowlistRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AllowlistRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	idint64 := state.ID.ValueInt64()
	_, err := retryOnRateLimit(ctx, func() (interface{}, error) {
		return nil, r.client.DeleteAllowlistRule(ctx, idint64)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Deleting Allowlist Rule",
			fmt.Sprintf("Unable to delete Allowlist Rule (id: %d), got error: %s", idint64, err),
		)
		return
	}
}

func (r *AllowlistRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data AllowlistRuleResourceModel

	id := req.ID
	idInt64, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing AllowlistRule",
			fmt.Sprintf("Unable to read AllowlistRule, got error: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idInt64)...)

	o, err := r.client.GetAllowlistRule(ctx, idInt64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Importing AllowlistRule",
			fmt.Sprintf("Unable to read AllowlistRule, got error: %s", err),
		)
		return
	}

	data = AllowlistRuleResourceModel{
		ID: types.Int64Value(idInt64),
		Ip: types.StringValue(o.Ip),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func validateAllowlistRule(_ *AllowlistRuleResourceModel) error {
	return nil
}
