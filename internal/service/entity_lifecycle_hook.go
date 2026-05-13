package service

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/sambeau/kanbanzai/internal/model"
	"github.com/sambeau/kanbanzai/internal/validate"
)

// EntityLifecycleHook allows DocumentService to trigger entity lifecycle
// transitions and update document reference fields on entities.
type EntityLifecycleHook interface {
	TransitionStatus(entityID, newStatus string) error
	SetDocumentRef(entityID, docField, docID string) error
	GetEntityStatus(entityID string) (entityType, status string, err error)
}

type entityLifecycleHookImpl struct {
	entitySvc *EntityService
}

// NewEntityLifecycleHook creates a hook that bridges DocumentService to EntityService.
func NewEntityLifecycleHook(entitySvc *EntityService) EntityLifecycleHook {
	return &entityLifecycleHookImpl{entitySvc: entitySvc}
}

func (h *entityLifecycleHookImpl) resolveEntity(entityID string) (entityType, resolvedID, slug string, err error) {
	if model.IsPlanID(entityID) {
		prefix, _, planSlug := model.ParsePlanID(entityID)
		switch prefix {
		case "P":
			return "strategic-plan", entityID, planSlug, nil
		case "B":
			return "plan", entityID, planSlug, nil
		default:
			// Unrecognized prefix — treat as batch for backward compatibility.
			return "plan", entityID, planSlug, nil
		}
	}
	if strings.HasPrefix(entityID, "FEAT-") {
		resolvedID, resolvedSlug, err := h.entitySvc.ResolvePrefix("feature", entityID)
		if err != nil {
			return "", "", "", fmt.Errorf("resolve feature %s: %w", entityID, err)
		}
		return "feature", resolvedID, resolvedSlug, nil
	}
	return "", "", "", fmt.Errorf("unsupported entity type for lifecycle hook: %s", entityID)
}

func (h *entityLifecycleHookImpl) TransitionStatus(entityID, newStatus string) error {
	entityType, resolvedID, slug, err := h.resolveEntity(entityID)
	if err != nil {
		return err
	}

	var kind validate.EntityKind
	var currentStatus string

	switch entityType {
	case "plan":
		result, err := h.entitySvc.GetPlan(resolvedID)
		if err != nil {
			return fmt.Errorf("load plan %s: %w", resolvedID, err)
		}
		currentStatus = stringFromState(result.State, "status")
		kind = validate.EntityPlan
	case "strategic-plan":
		result, err := h.entitySvc.GetStrategicPlan(resolvedID)
		if err != nil {
			return fmt.Errorf("load strategic plan %s: %w", resolvedID, err)
		}
		currentStatus = stringFromState(result.State, "status")
		kind = validate.EntityStrategicPlan
	case "feature":
		result, err := h.entitySvc.Get(context.Background(), "feature", resolvedID, slug)
		if err != nil {
			return fmt.Errorf("load feature %s: %w", resolvedID, err)
		}
		currentStatus = stringFromState(result.State, "status")
		kind = validate.EntityFeature
	}

	if !validate.CanTransition(kind, currentStatus, newStatus) {
		log.Printf("lifecycle hook: skipping transition %s %s -> %s (not valid from current state %q)", entityType, resolvedID, newStatus, currentStatus)
		return nil
	}

	switch entityType {
	case "plan":
		_, err = h.entitySvc.UpdatePlanStatus(resolvedID, slug, newStatus)
	case "strategic-plan":
		_, err = h.entitySvc.UpdateStrategicPlanStatus(resolvedID, slug, newStatus)
	case "feature":
		_, err = h.entitySvc.UpdateStatus(UpdateStatusInput{
			Type:   "feature",
			ID:     resolvedID,
			Slug:   slug,
			Status: newStatus,
		})
	}
	return err
}

func (h *entityLifecycleHookImpl) SetDocumentRef(entityID, docField, docID string) error {
	entityType, resolvedID, slug, err := h.resolveEntity(entityID)
	if err != nil {
		return err
	}

	switch entityType {
	case "plan":
		if docField != "design" {
			return nil
		}
		designVal := docID
		_, err := h.entitySvc.UpdatePlan(UpdatePlanInput{
			ID:     resolvedID,
			Slug:   slug,
			Design: &designVal,
		})
		return err
	case "strategic-plan":
		if docField != "design" {
			return nil
		}
		designVal := docID
		_, err := h.entitySvc.UpdateStrategicPlan(UpdateStrategicPlanInput{
			ID:     resolvedID,
			Slug:   slug,
			Design: &designVal,
		})
		return err
	case "feature":
		fields := map[string]string{docField: docID}
		_, err := h.entitySvc.UpdateEntity(UpdateEntityInput{
			Type:   "feature",
			ID:     resolvedID,
			Slug:   slug,
			Fields: fields,
		})
		return err
	}
	return nil
}

func (h *entityLifecycleHookImpl) GetEntityStatus(entityID string) (string, string, error) {
	entityType, resolvedID, slug, err := h.resolveEntity(entityID)
	if err != nil {
		return "", "", err
	}

	switch entityType {
	case "plan":
		result, err := h.entitySvc.GetPlan(resolvedID)
		if err != nil {
			return "", "", err
		}
		return "plan", stringFromState(result.State, "status"), nil
	case "strategic-plan":
		result, err := h.entitySvc.GetStrategicPlan(resolvedID)
		if err != nil {
			return "", "", err
		}
		return "strategic-plan", stringFromState(result.State, "status"), nil
	case "feature":
		result, err := h.entitySvc.Get(context.Background(), "feature", resolvedID, slug)
		if err != nil {
			return "", "", err
		}
		return "feature", stringFromState(result.State, "status"), nil
	}
	return "", "", fmt.Errorf("unsupported entity type: %s", entityType)
}
