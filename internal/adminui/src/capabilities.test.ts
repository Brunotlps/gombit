import { describe, expect, it } from "vitest";

import type { ModelMeta } from "./api/types";
import {
  canCreate,
  canDelete,
  canList,
  canUpdate,
  canViewDetail,
} from "./capabilities";

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
    create: true,
    update: true,
    delete: true,
  },
};

function withCan(can: Partial<ModelMeta["can"]>): ModelMeta {
  return { ...model, can: { ...model.can, ...can } };
}

function withActions(actions: Partial<ModelMeta["actions"]>): ModelMeta {
  return { ...model, actions: { ...model.actions, ...actions } };
}

describe("canList", () => {
  it("hides the list when can.view is false", () => {
    expect(canList(withCan({ view: false }))).toBe(false);
  });

  it("hides the list when actions.list is false even when can.view is true", () => {
    expect(canList(withActions({ list: false }))).toBe(false);
  });
});

describe("canViewDetail", () => {
  it("hides the detail action when can.view is false", () => {
    expect(canViewDetail(withCan({ view: false }))).toBe(false);
  });

  it("hides the detail action when actions.detail is false even when can.view is true", () => {
    expect(canViewDetail(withActions({ detail: false }))).toBe(false);
  });
});

describe("canCreate", () => {
  it("hides the create action when can.create is false", () => {
    expect(canCreate(withCan({ create: false }))).toBe(false);
  });

  it("hides the create action when actions.create is false even when can.create is true", () => {
    expect(canCreate(withActions({ create: false }))).toBe(false);
  });
});

describe("canUpdate", () => {
  it("hides the update action when can.update is false", () => {
    expect(canUpdate(withCan({ update: false }))).toBe(false);
  });

  it("hides the update action when actions.update is false even when can.update is true", () => {
    expect(canUpdate(withActions({ update: false }))).toBe(false);
  });
});

describe("canDelete", () => {
  it("hides the delete action when can.delete is false", () => {
    expect(canDelete(withCan({ delete: false }))).toBe(false);
  });

  it("hides the delete action when actions.delete is false even when can.delete is true", () => {
    expect(canDelete(withActions({ delete: false }))).toBe(false);
  });
});
