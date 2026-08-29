import { describe, expect, it } from "vitest";

import type { FieldMeta } from "./api/types";
import { formValuesToBody } from "./fields";

function field(partial: Pick<FieldMeta, "name" | "type"> & Partial<FieldMeta>): FieldMeta {
  return {
    required: false,
    readonly: false,
    ...partial,
  };
}

describe("formValuesToBody", () => {
  it("includes emptied optional string/text/date/datetime/json as null", () => {
    const fields: FieldMeta[] = [
      field({ name: "note", type: "string" }),
      field({ name: "body", type: "text" }),
      field({ name: "due", type: "date" }),
      field({ name: "published_at", type: "datetime" }),
      field({ name: "payload", type: "json" }),
    ];
    const { body, jsonErrors } = formValuesToBody(
      { note: "", body: null, due: undefined, published_at: "", payload: "  " },
      fields,
    );
    expect(jsonErrors).toEqual({});
    expect(body).toEqual({
      note: null,
      body: null,
      due: null,
      published_at: null,
      payload: null,
    });
  });

  it("includes required empties as null so the server can 422", () => {
    const fields: FieldMeta[] = [field({ name: "name", type: "string", required: true })];
    const { body, jsonErrors } = formValuesToBody({ name: "" }, fields);
    expect(jsonErrors).toEqual({});
    expect(body).toEqual({ name: null });
  });

  it("sends null for emptied optional numbers and keeps 0", () => {
    const fields: FieldMeta[] = [
      field({ name: "price", type: "integer" }),
      field({ name: "weight", type: "float" }),
      field({ name: "qty", type: "decimal" }),
    ];
    const cleared = formValuesToBody({ price: "", weight: null, qty: undefined }, fields);
    expect(cleared.jsonErrors).toEqual({});
    expect(cleared.body).toEqual({ price: null, weight: null, qty: null });

    const zeros = formValuesToBody({ price: 0, weight: "0", qty: 0 }, fields);
    expect(zeros.jsonErrors).toEqual({});
    expect(zeros.body).toEqual({ price: 0, weight: 0, qty: 0 });
  });

  it("always sends booleans, including false", () => {
    const fields: FieldMeta[] = [field({ name: "active", type: "boolean" })];
    const { body, jsonErrors } = formValuesToBody({ active: false }, fields);
    expect(jsonErrors).toEqual({});
    expect(body).toEqual({ active: false });
  });

  it("keeps invalid JSON as jsonErrors and omits that key", () => {
    const fields: FieldMeta[] = [
      field({ name: "payload", type: "json" }),
      field({ name: "note", type: "string" }),
    ];
    const { body, jsonErrors } = formValuesToBody({ payload: "{", note: "" }, fields);
    expect(jsonErrors).toEqual({ payload: "must be valid JSON" });
    expect(body).toEqual({ note: null });
    expect(body).not.toHaveProperty("payload");
  });

  it("does not put readonly fields in the body", () => {
    const fields: FieldMeta[] = [
      field({ name: "id", type: "integer", readonly: true }),
      field({ name: "note", type: "text" }),
    ];
    const { body } = formValuesToBody({ id: 3, note: "" }, fields);
    expect(body).toEqual({ note: null });
  });
});

describe("many_to_many fields", () => {
  const rel = (): FieldMeta => ({
    name: "warehouses",
    type: "relation",
    required: false,
    readonly: false,
    related: { slug: "warehouses", kind: "many_to_many", label_field: "name" },
  });

  it("submits the selected ids as a numeric list", () => {
    const { body } = formValuesToBody({ warehouses: ["1", "2", "3"] }, [rel()]);
    expect(body.warehouses).toEqual([1, 2, 3]);
  });

  it("submits an empty list to clear the relation", () => {
    const { body } = formValuesToBody({ warehouses: [] }, [rel()]);
    expect(body.warehouses).toEqual([]);
  });

  it("preserves string / uuid ids instead of dropping them", () => {
    const uuid = "6f9619ff-8b86-d011-b42d-00c04fc964ff";
    const { body } = formValuesToBody({ warehouses: [uuid, "abc"] }, [rel()]);
    expect(body.warehouses).toEqual([uuid, "abc"]);
  });

  it("drops only null / undefined / empty entries", () => {
    const { body } = formValuesToBody({ warehouses: ["4", "", null, 5] }, [rel()]);
    expect(body.warehouses).toEqual([4, 5]);
  });
});
