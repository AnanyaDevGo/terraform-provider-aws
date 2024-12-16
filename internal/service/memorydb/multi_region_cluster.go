// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package memorydb

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	awstypes "github.com/aws/aws-sdk-go-v2/service/memorydb/types"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/enum"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	"github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @FrameworkResource("aws_memorydb_multi_region_cluster", name="Multi Region Cluster")
// @Tags(identifierAttribute="arn")
func newMultiRegionClusterResource(_ context.Context) (resource.ResourceWithConfigure, error) {
	r := &multiRegionClusterResource{}

	r.SetDefaultCreateTimeout(120 * time.Minute)
	r.SetDefaultUpdateTimeout(120 * time.Minute)
	r.SetDefaultDeleteTimeout(120 * time.Minute)

	return r, nil
}

const (
	ResNameMultiRegionCluster = "Multi Region Cluster"
)

type multiRegionClusterResource struct {
	framework.ResourceWithConfigure
	framework.WithTimeouts
}

func (*multiRegionClusterResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "aws_memorydb_multi_region_cluster"
}

func (r *multiRegionClusterResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrDescription: schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Default: stringdefault.StaticString("Managed by Terraform"),
			},
			names.AttrEngine: schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					enum.FrameworkValidate[clusterEngine](),
				},
			},
			names.AttrEngineVersion: schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrID: framework.IDAttribute(),
			"multi_region_cluster_name": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"multi_region_cluster_name_suffix": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"multi_region_parameter_group_name": schema.StringAttribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node_type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"num_shards": schema.Int64Attribute{
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			names.AttrStatus: schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
			"tls_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"update_strategy": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					enum.FrameworkValidate[awstypes.UpdateStrategy](),
				},
			},
		},
		Blocks: map[string]schema.Block{
			names.AttrTimeouts: timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *multiRegionClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	conn := r.Meta().MemoryDBClient(ctx)

	var plan multiRegionClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var input memorydb.CreateMultiRegionClusterInput
	resp.Diagnostics.Append(flex.Expand(ctx, plan, &input)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input.Tags = getTagsIn(ctx)

	out, err := conn.CreateMultiRegionCluster(ctx, &input)
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.MemoryDB, create.ErrActionCreating, ResNameMultiRegionCluster, plan.MultiRegionClusterName.String(), err),
			err.Error(),
		)
		return
	}
	if out == nil || out.MultiRegionCluster == nil || out.MultiRegionCluster.ARN == nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.MemoryDB, create.ErrActionCreating, ResNameMultiRegionCluster, plan.MultiRegionClusterName.String(), nil),
			errors.New("empty output").Error(),
		)
		return
	}

	plan.ID = flex.StringToFramework(ctx, out.MultiRegionCluster.MultiRegionClusterName)
	plan.NumShards = flex.Int32ToFramework(ctx, out.MultiRegionCluster.NumberOfShards)

	createTimeout := r.CreateTimeout(ctx, plan.Timeouts)
	statusOut, err := waitMultiRegionClusterAvailable(ctx, conn, plan.ID.ValueString(), createTimeout)
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.MemoryDB, create.ErrActionWaitingForCreation, ResNameMultiRegionCluster, plan.MultiRegionClusterName.String(), err),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(flex.Flatten(ctx, statusOut, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *multiRegionClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	conn := r.Meta().MemoryDBClient(ctx)

	var state multiRegionClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := findMultiRegionClusterByName(ctx, conn, state.ID.ValueString())
	if tfresource.NotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.MemoryDB, create.ErrActionReading, ResNameMultiRegionCluster, state.ID.String(), err),
			err.Error(),
		)
		return
	}

	suffix, err := suffixAfterHyphen(aws.ToString(out.MultiRegionClusterName))
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.MemoryDB, create.ErrActionSetting, ResNameMultiRegionCluster, state.ID.String(), err),
			err.Error(),
		)
		return
	}
	state.MultiRegionClusterNameSuffix = flex.StringToFramework(ctx, &suffix)
	state.NumShards = flex.Int32ToFramework(ctx, out.NumberOfShards)

	resp.Diagnostics.Append(flex.Flatten(ctx, out, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *multiRegionClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	conn := r.Meta().MemoryDBClient(ctx)

	var plan, state multiRegionClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.MultiRegionParameterGroupName.Equal(state.MultiRegionParameterGroupName) ||
		!plan.NodeType.Equal(state.NodeType) ||
		!plan.NumShards.Equal(state.NumShards) {
		input := memorydb.UpdateMultiRegionClusterInput{
			MultiRegionClusterName: state.MultiRegionClusterName.ValueStringPointer(),
		}

		if !plan.MultiRegionParameterGroupName.Equal(state.MultiRegionParameterGroupName) {
			input.MultiRegionParameterGroupName = plan.MultiRegionParameterGroupName.ValueStringPointer()
		}

		if !plan.NodeType.Equal(state.NodeType) {
			input.NodeType = plan.NodeType.ValueStringPointer()
		}

		if !plan.NumShards.Equal(state.NumShards) {
			input.ShardConfiguration = &awstypes.ShardConfigurationRequest{
				ShardCount: int32(*plan.NumShards.ValueInt64Pointer()),
			}
		}

		if !plan.UpdateStrategy.Equal(state.UpdateStrategy) {
			input.UpdateStrategy = awstypes.UpdateStrategy(plan.UpdateStrategy.ValueString())
		}

		resp.Diagnostics.Append(flex.Expand(ctx, plan, &input)...)
		if resp.Diagnostics.HasError() {
			return
		}

		_, err := conn.UpdateMultiRegionCluster(ctx, &input)
		if err != nil {
			resp.Diagnostics.AddError(
				create.ProblemStandardMessage(names.MemoryDB, create.ErrActionUpdating, ResNameMultiRegionCluster, plan.MultiRegionClusterName.String(), err),
				err.Error(),
			)
			return
		}

		updateTimeout := r.UpdateTimeout(ctx, plan.Timeouts)
		statusOut, err := waitMultiRegionClusterAvailable(ctx, conn, plan.ID.ValueString(), updateTimeout)
		if err != nil {
			resp.Diagnostics.AddError(
				create.ProblemStandardMessage(names.MemoryDB, create.ErrActionWaitingForUpdate, ResNameMultiRegionCluster, plan.MultiRegionClusterName.String(), err),
				err.Error(),
			)
			return
		}

		resp.Diagnostics.Append(flex.Flatten(ctx, statusOut, &plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *multiRegionClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	conn := r.Meta().MemoryDBClient(ctx)

	var state multiRegionClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Before deleting the multi-region cluster, ensure it is ready for deletion.
	// Removing an `aws_memorydb_cluster` from a multi-region cluster may temporarily block deletion.
	createTimeout := r.CreateTimeout(ctx, state.Timeouts)
	_, err := waitMultiRegionClusterAvailable(ctx, conn, state.ID.ValueString(), createTimeout)
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.MemoryDB, create.ErrActionWaitingForCreation, ResNameMultiRegionCluster, state.MultiRegionClusterName.String(), err),
			err.Error(),
		)
		return
	}

	input := memorydb.DeleteMultiRegionClusterInput{
		MultiRegionClusterName: state.MultiRegionClusterName.ValueStringPointer(),
	}

	_, err = conn.DeleteMultiRegionCluster(ctx, &input)
	if err != nil {
		if errs.IsA[*awstypes.MultiRegionClusterNotFoundFault](err) {
			return
		}
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.MemoryDB, create.ErrActionDeleting, ResNameMultiRegionCluster, state.MultiRegionClusterName.String(), err),
			err.Error(),
		)
		return
	}

	deleteTimeout := r.DeleteTimeout(ctx, state.Timeouts)
	_, err = waitMultiRegionClusterDeleted(ctx, conn, state.ID.ValueString(), deleteTimeout)
	if err != nil {
		resp.Diagnostics.AddError(
			create.ProblemStandardMessage(names.MemoryDB, create.ErrActionWaitingForDeletion, ResNameMultiRegionCluster, state.MultiRegionClusterName.String(), err),
			err.Error(),
		)
		return
	}
}

func (r *multiRegionClusterResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(names.AttrID), request, response)
}

func (r *multiRegionClusterResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	r.SetTagsAll(ctx, req, resp)
}

type multiRegionClusterResourceModel struct {
	ARN                           types.String   `tfsdk:"arn"`
	Description                   types.String   `tfsdk:"description"`
	Engine                        types.String   `tfsdk:"engine"`
	EngineVersion                 types.String   `tfsdk:"engine_version"`
	ID                            types.String   `tfsdk:"id"`
	MultiRegionClusterName        types.String   `tfsdk:"multi_region_cluster_name"`
	MultiRegionClusterNameSuffix  types.String   `tfsdk:"multi_region_cluster_name_suffix"`
	MultiRegionParameterGroupName types.String   `tfsdk:"multi_region_parameter_group_name"`
	NodeType                      types.String   `tfsdk:"node_type"`
	NumShards                     types.Int64    `tfsdk:"num_shards"`
	Status                        types.String   `tfsdk:"status"`
	Tags                          tftags.Map     `tfsdk:"tags"`
	TagsAll                       tftags.Map     `tfsdk:"tags_all"`
	Timeouts                      timeouts.Value `tfsdk:"timeouts"`
	TLSEnabled                    types.Bool     `tfsdk:"tls_enabled"`
	UpdateStrategy                types.String   `tfsdk:"update_strategy"`
}

func findMultiRegionClusterByName(ctx context.Context, conn *memorydb.Client, name string) (*awstypes.MultiRegionCluster, error) {
	input := &memorydb.DescribeMultiRegionClustersInput{
		MultiRegionClusterName: aws.String(name),
		ShowClusterDetails:     aws.Bool(true),
	}

	return findMultiRegionCluster(ctx, conn, input)
}

func findMultiRegionCluster(ctx context.Context, conn *memorydb.Client, input *memorydb.DescribeMultiRegionClustersInput) (*awstypes.MultiRegionCluster, error) {
	output, err := findMultiRegionClusters(ctx, conn, input)

	if err != nil {
		return nil, err
	}

	return tfresource.AssertSingleValueResult(output)
}

func findMultiRegionClusters(ctx context.Context, conn *memorydb.Client, input *memorydb.DescribeMultiRegionClustersInput) ([]awstypes.MultiRegionCluster, error) {
	var output []awstypes.MultiRegionCluster

	pages := memorydb.NewDescribeMultiRegionClustersPaginator(conn, input)
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)

		if errs.IsA[*awstypes.MultiRegionClusterNotFoundFault](err) {
			return nil, &retry.NotFoundError{
				LastError:   err,
				LastRequest: input,
			}
		}

		if err != nil {
			return nil, err
		}

		output = append(output, page.MultiRegionClusters...)
	}

	return output, nil
}

func statusMultiRegionCluster(ctx context.Context, conn *memorydb.Client, name string) retry.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := findMultiRegionClusterByName(ctx, conn, name)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.ToString(output.Status), nil
	}
}

func waitMultiRegionClusterAvailable(ctx context.Context, conn *memorydb.Client, name string, timeout time.Duration) (*awstypes.MultiRegionCluster, error) {
	stateConf := &retry.StateChangeConf{
		Delay:                     20 * time.Second,
		Pending:                   []string{clusterStatusCreating, clusterStatusUpdating, clusterStatusSnapshotting},
		Target:                    []string{clusterStatusAvailable},
		Refresh:                   statusMultiRegionCluster(ctx, conn, name),
		ContinuousTargetOccurence: 3,
		Timeout:                   timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*awstypes.MultiRegionCluster); ok {
		return output, err
	}

	return nil, err
}

func waitMultiRegionClusterDeleted(ctx context.Context, conn *memorydb.Client, name string, timeout time.Duration) (*awstypes.MultiRegionCluster, error) {
	stateConf := &retry.StateChangeConf{
		Delay:                     20 * time.Second,
		Pending:                   []string{clusterStatusDeleting},
		Target:                    []string{},
		Refresh:                   statusMultiRegionCluster(ctx, conn, name),
		ContinuousTargetOccurence: 3,
		Timeout:                   timeout,
	}

	outputRaw, err := stateConf.WaitForStateContext(ctx)

	if output, ok := outputRaw.(*awstypes.MultiRegionCluster); ok {
		return output, err
	}

	return nil, err
}

// suffixAfterHyphen extracts the substring after the first hyphen ("-") in the input string.
// If no hyphen is found, it returns an error.
func suffixAfterHyphen(input string) (string, error) {
	idx := strings.Index(input, "-")
	if idx == -1 {
		return "", errors.New("no hyphen found in the input string")
	}
	return input[idx+1:], nil
}
