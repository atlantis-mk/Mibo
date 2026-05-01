package metadata

import (
	"context"
	"fmt"
)

func (s *Service) runMetadataOperation(ctx context.Context, input MetadataOperationRequest) (MetadataOperationResult, error) {
	if input.OriginItemID != 0 && input.TargetItemID == 0 {
		target, err := s.resolveCatalogMatchTarget(ctx, input.OriginItemID)
		if err != nil {
			return MetadataOperationResult{}, err
		}
		input.TargetItemID = target.ID
	}
	switch input.Operation {
	case OperationTypeMatch:
		result, ok, err := s.runMatchMetadataOperation(ctx, input)
		if err != nil {
			return MetadataOperationResult{}, err
		}
		if !ok {
			return MetadataOperationResult{}, fmt.Errorf("metadata match does not support item %d", input.OriginItemID)
		}
		return result, nil
	case OperationTypeRefetch:
		result, ok, err := s.runRefetchMetadataOperation(ctx, input)
		if err != nil {
			return MetadataOperationResult{}, err
		}
		if !ok {
			return MetadataOperationResult{}, fmt.Errorf("metadata refetch does not support item %d", input.OriginItemID)
		}
		return result, nil
	case OperationTypeManualApply:
		result, err := s.runManualApplyMetadataOperation(ctx, input)
		if err != nil {
			return MetadataOperationResult{}, err
		}
		return result, nil
	default:
		return MetadataOperationResult{}, fmt.Errorf("unsupported metadata operation %q", input.Operation)
	}
}
