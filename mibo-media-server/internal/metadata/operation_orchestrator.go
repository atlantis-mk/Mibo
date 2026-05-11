package metadata

import (
	"context"
	"fmt"
)

func (s *Service) runMetadataOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, error) {
	if input.OriginMetadataItemID == 0 && input.TargetMetadataItemID == 0 {
		return MetadataOperationResult{}, fmt.Errorf("metadata item id is required")
	}
	return s.runMetadataItemOperation(ctx, input)
}

func (s *Service) runMetadataItemOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, error) {
	if input.TargetMetadataItemID == 0 {
		input.TargetMetadataItemID = input.OriginMetadataItemID
	}
	if input.OriginMetadataItemID == 0 {
		input.OriginMetadataItemID = input.TargetMetadataItemID
	}
	switch input.Operation {
	case OperationTypeMatch:
		result, ok, err := s.runMatchMetadataItemOperation(ctx, input)
		if err != nil {
			return MetadataOperationResult{}, err
		}
		if !ok {
			return MetadataOperationResult{}, fmt.Errorf("metadata match does not support metadata item %d", input.OriginMetadataItemID)
		}
		return result, nil
	case OperationTypeRefetch:
		result, ok, err := s.runRefetchMetadataItemOperation(ctx, input)
		if err != nil {
			return MetadataOperationResult{}, err
		}
		if !ok {
			return MetadataOperationResult{}, fmt.Errorf("metadata refetch does not support metadata item %d", input.OriginMetadataItemID)
		}
		return result, nil
	case OperationTypeManualApply:
		return s.runManualApplyMetadataItemOperation(ctx, input)
	default:
		return MetadataOperationResult{}, fmt.Errorf("unsupported metadata operation %q", input.Operation)
	}
}
