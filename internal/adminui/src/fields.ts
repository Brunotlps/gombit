import type { FieldMeta, Row } from "./api/types";

export function isHasMany(field: FieldMeta): boolean {
  return field.type === "relation" && field.related?.kind === "has_many";
}

export function isWritable(field: FieldMeta): boolean {
  return !field.readonly && !isHasMany(field);
}

export function writableFields(fields: FieldMeta[]): FieldMeta[] {
  return fields.filter(isWritable);
}

export function formatCell(value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "boolean") {
    return value ? "yes" : "no";
  }
  if (typeof value === "object") {
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }
  return String(value);
}

export function emptyFormValue(field: FieldMeta): unknown {
  if (field.type === "boolean") {
    return false;
  }
  if (field.type === "json") {
    return "";
  }
  return "";
}

export function rowToFormValues(row: Row, fields: FieldMeta[]): Row {
  const values: Row = {};
  for (const field of fields) {
    const raw = row[field.name];
    values[field.name] = valueToForm(field, raw);
  }
  return values;
}

export function formValuesToBody(values: Row, fields: FieldMeta[]): { body: Row; jsonErrors: Record<string, string> } {
  const body: Row = {};
  const jsonErrors: Record<string, string> = {};
  for (const field of writableFields(fields)) {
    const raw = values[field.name];
    if (field.type === "json") {
      const text = raw === undefined || raw === null ? "" : String(raw).trim();
      if (text === "") {
        continue;
      }
      try {
        body[field.name] = JSON.parse(text) as unknown;
      } catch {
        jsonErrors[field.name] = "must be valid JSON";
      }
      continue;
    }
    if (field.type === "boolean") {
      body[field.name] = Boolean(raw);
      continue;
    }
    if (field.type === "integer" || field.type === "float" || field.type === "decimal") {
      if (raw === "" || raw === undefined || raw === null) {
        continue;
      }
      const n = Number(raw);
      if (Number.isNaN(n)) {
        jsonErrors[field.name] = "must be a number";
        continue;
      }
      body[field.name] = field.type === "integer" ? Math.trunc(n) : n;
      continue;
    }
    if (raw === "" || raw === undefined || raw === null) {
      continue;
    }
    body[field.name] = raw;
  }
  return { body, jsonErrors };
}

function valueToForm(field: FieldMeta, raw: unknown): unknown {
  if (field.type === "boolean") {
    return Boolean(raw);
  }
  if (field.type === "json") {
    if (raw === undefined || raw === null || raw === "") {
      return "";
    }
    if (typeof raw === "string") {
      return raw;
    }
    try {
      return JSON.stringify(raw, null, 2);
    } catch {
      return String(raw);
    }
  }
  if (field.type === "datetime" && typeof raw === "string") {
    return datetimeLocalValue(raw);
  }
  if (field.type === "date" && typeof raw === "string") {
    return raw.slice(0, 10);
  }
  if (raw === undefined || raw === null) {
    return emptyFormValue(field);
  }
  return raw;
}

function datetimeLocalValue(raw: string): string {
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return raw.length >= 16 ? raw.slice(0, 16) : raw;
  }
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}
