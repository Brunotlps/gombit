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
    if (field.type === "datetime") {
      body[field.name] = datetimeLocalToRFC3339(String(raw));
      continue;
    }
    if (field.type === "date") {
      // ADMIN-1 asDate expects YYYY-MM-DD; datetime-local is out of band here.
      body[field.name] = String(raw).trim().slice(0, 10);
      continue;
    }
    body[field.name] = raw;
  }
  return { body, jsonErrors };
}

/**
 * Convert a `datetime-local` value to RFC3339 for ADMIN-1 `asDateTime`.
 *
 * `<input type="datetime-local">` submits `YYYY-MM-DDTHH:mm` (optional
 * seconds, no offset). The Go converter only accepts RFC3339 / RFC3339Nano.
 * Wall time is treated as the browser's local zone; the result is ISO-8601
 * with a `Z` offset (`Date.toISOString()`). Values that already carry `Z`
 * or a numeric offset are left unchanged. Date-only fields stay YYYY-MM-DD
 * in `formValuesToBody` and never go through this helper.
 *
 * Examples:
 *   `2026-08-18T12:30`     → `2026-08-18T...Z` (local → UTC)
 *   `2026-08-18T12:30:00Z` → unchanged
 */
export function datetimeLocalToRFC3339(raw: string): string {
  const trimmed = raw.trim();
  if (trimmed === "") {
    return trimmed;
  }
  if (/[Zz]$/.test(trimmed) || /[+-]\d{2}:\d{2}$/.test(trimmed)) {
    return trimmed;
  }
  const match = trimmed.match(
    /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2})(?:\.(\d+))?)?$/,
  );
  if (match) {
    const year = Number(match[1]);
    const month = Number(match[2]);
    const day = Number(match[3]);
    const hour = Number(match[4]);
    const minute = Number(match[5]);
    const second = match[6] ? Number(match[6]) : 0;
    const millis = match[7] ? Number(match[7].slice(0, 3).padEnd(3, "0")) : 0;
    const date = new Date(year, month - 1, day, hour, minute, second, millis);
    if (!Number.isNaN(date.getTime())) {
      return date.toISOString();
    }
    const ss = String(second).padStart(2, "0");
    return `${match[1]}-${match[2]}-${match[3]}T${match[4]}:${match[5]}:${ss}Z`;
  }
  const parsed = new Date(trimmed);
  if (!Number.isNaN(parsed.getTime())) {
    return parsed.toISOString();
  }
  const withSeconds = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(trimmed) ? `${trimmed}:00` : trimmed;
  return withSeconds.endsWith("Z") ? withSeconds : `${withSeconds}Z`;
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
