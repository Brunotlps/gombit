import type { ModelMeta } from "./api/types";

export function canList(model: ModelMeta): boolean {
  return model.actions.list && model.can.view;
}

export function canViewDetail(model: ModelMeta): boolean {
  return model.actions.detail && model.can.view;
}

export function canCreate(model: ModelMeta): boolean {
  return model.actions.create && model.can.create;
}

export function canUpdate(model: ModelMeta): boolean {
  return model.actions.update && model.can.update;
}

// Edit hydrates via GET /resources/{slug}/{id}, which ADMIN-1 gates on
// actions.detail + can.view. Without a loaded row, empty boolean defaults
// (false) would PATCH over stored true values.
export function canPopulateEditForm(model: ModelMeta): boolean {
  return canUpdate(model) && canViewDetail(model);
}

export function canDelete(model: ModelMeta): boolean {
  return model.actions.delete && model.can.delete;
}
