import { describe, expect, it } from "vitest";

import type { ModelMeta } from "./api/types";
import { canCreate } from "./capabilities";

const model: ModelMeta = {
  slug: "widgets",
  singular: "Widget",
  plural: "Widgets",
  pk: "id",
  fields: [],
  list: [],
  search: [],
  filter: [],
  ordering: [],
  actions: {
    list: true,
    detail: true,
    create: true,
    update: true,
    delete: true,
  },
  permissions: {
    view: "admin.widgets.view",
    create: "admin.widgets.create",
    update: "admin.widgets.update",
    delete: "admin.widgets.delete",
  },
  can: {
    view: true,
    create: false,
    update: false,
    delete: false,
  },
};

describe("canCreate", () => {
  it("hides the New action when the user cannot create", () => {
    expect(canCreate(model)).toBe(false);
  });

  it("hides the New action when create is disabled", () => {
    expect(canCreate({ ...model, actions: { ...model.actions, create: false }, can: { ...model.can, create: true } })).toBe(false);
  });
});
