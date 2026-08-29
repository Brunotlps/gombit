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

    // integer/float stay JSON numbers; decimal is an exact string (never a
    // float, which the data plane rejects for a decimal column).
    const zeros = formValuesToBody({ price: 0, weight: "0", qty: 0 }, fields);
    expect(zeros.jsonErrors).toEqual({});
    expect(zeros.body).toEqual({ price: 0, weight: 0, qty: "0" });
  });

  it("sends decimal as an exact string and rejects non-numeric text", () => {
    const fields: FieldMeta[] = [field({ name: "amount", type: "decimal" })];
    const ok = formValuesToBody({ amount: "19.9999" }, fields);
    expect(ok.jsonErrors).toEqual({});
    expect(ok.body).toEqual({ amount: "19.9999" });

    const bad = formValuesToBody({ amount: "1.2.3" }, fields);
    expect(bad.jsonErrors).toHaveProperty("amount");
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
