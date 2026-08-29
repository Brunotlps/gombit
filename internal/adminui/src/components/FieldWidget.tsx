import { useEffect, useState } from "react";
import { Controller, type Control, type FieldValues } from "react-hook-form";

import { Checkbox, FormControlLabel, MenuItem, TextField } from "@mui/material";

import { useApiClient } from "../api/client";
import { useCatalog } from "../app/providers";
import type { FieldMeta } from "../api/types";
import { isBelongsTo, isHasMany, isManyToMany, relationOptions, toIdList, toRelId } from "../fields";

type Props = {
  field: FieldMeta;
  control: Control<FieldValues>;
  disabled?: boolean;
};

type Option = { value: string; label: string };

// relationPageSize is the related-model page fetched for the picker. It is
// contract.MaxPerPage; related rows beyond it are not selectable from the
// dropdown yet (a searchable/paged picker is a follow-up). The widget flags
// when the list is truncated.
const relationPageSize = 100;

export function FieldWidget({ field, control, disabled }: Props) {
  const readOnly = disabled || field.readonly || isHasMany(field);
  const label = fieldLabel(field);

  if (isManyToMany(field)) {
    return <RelationMultiSelect field={field} control={control} disabled={disabled} />;
  }

  if (isBelongsTo(field)) {
    return <RelationSelect field={field} control={control} disabled={disabled} />;
  }

  if (field.type === "boolean") {
    return (
      <Controller
        name={field.name}
        control={control}
        render={({ field: rhf }) => (
          <FormControlLabel
            control={
              <Checkbox
                checked={Boolean(rhf.value)}
                onChange={(event) => rhf.onChange(event.target.checked)}
                disabled={readOnly}
              />
            }
            label={label}
          />
        )}
      />
    );
  }

  const inputType = inputTypeFor(field);
  const multiline = field.type === "text" || field.type === "json";

  return (
    <Controller
      name={field.name}
      control={control}
      rules={{ required: field.required && !readOnly }}
      render={({ field: rhf, fieldState }) => (
        <TextField
          {...rhf}
          value={rhf.value ?? ""}
          type={inputType}
          label={label}
          fullWidth
          multiline={multiline}
          minRows={multiline ? 3 : undefined}
          error={!!fieldState.error}
          helperText={fieldState.error?.message ?? helperText(field)}
          disabled={readOnly}
          slotProps={
            field.type === "datetime" || field.type === "date"
              ? { inputLabel: { shrink: true } }
              : undefined
          }
        />
      )}
    />
  );
}

/**
 * useRelationOptions loads the related model's rows (via the data-plane list
 * endpoint — client.list -> /api/v1/admin/...; a bare /admin/{slug} hits the SPA
 * embed fallback) and maps them to { value: pk, label } options.
 */
function useRelationOptions(field: FieldMeta): {
  options: Option[];
  helper: string;
  loadError: string;
} {
  const client = useApiClient();
  const { bySlug } = useCatalog();
  const slug = field.related?.slug ?? "";
  const labelField = field.related?.label_field ?? "";
  // The related model's primary key field (may be a uuid / string, not "id").
  const pkField = bySlug.get(slug)?.pk ?? "id";
  const [options, setOptions] = useState<Option[]>([]);
  const [truncated, setTruncated] = useState(false);
  const [loadError, setLoadError] = useState("");

  useEffect(() => {
    if (!slug) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const env = await client.list(slug, { per_page: relationPageSize });
        if (cancelled) {
          return;
        }
        const rows = Array.isArray(env.data) ? env.data : [];
        setTruncated((env.meta?.total ?? rows.length) > relationPageSize);
        setOptions(relationOptions(rows, pkField, labelField));
      } catch (err: unknown) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : "failed to load options");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [client, slug, labelField, pkField]);

  const helper = loadError
    ? loadError
    : truncated
      ? `Showing first ${relationPageSize} ${slug}; not all rows are selectable yet`
      : `Select related ${slug}`;
  return { options, helper, loadError };
}

/**
 * RelationMultiSelect renders a many-to-many field as a multi-select backed by
 * the related model's admin list endpoint (#223).
 */
function RelationMultiSelect({ field, control, disabled }: Props) {
  const { options, helper, loadError } = useRelationOptions(field);
  return (
    <Controller
      name={field.name}
      control={control}
      render={({ field: rhf }) => {
        const selected = Array.isArray(rhf.value) ? rhf.value.map((v) => String(v)) : [];
        return (
          <TextField
            select
            label={fieldLabel(field)}
            fullWidth
            disabled={disabled || field.readonly}
            value={selected}
            onChange={(event) => {
              const raw = event.target.value as unknown as string[] | string;
              const list = Array.isArray(raw) ? raw : [raw];
              // Preserve the related PK type (number vs uuid/string) via toIdList.
              rhf.onChange(toIdList(list));
            }}
            helperText={helper}
            error={!!loadError}
            slotProps={{ select: { multiple: true } }}
          >
            {options.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </TextField>
        );
      }}
    />
  );
}

/**
 * RelationSelect renders a belongs_to foreign key as a single-select backed by
 * the related model's list endpoint, storing the selected primary key (#223).
 */
function RelationSelect({ field, control, disabled }: Props) {
  const { options, helper, loadError } = useRelationOptions(field);
  const readOnly = disabled || field.readonly;
  return (
    <Controller
      name={field.name}
      control={control}
      rules={{ required: field.required && !readOnly }}
      render={({ field: rhf, fieldState }) => (
        <TextField
          select
          label={fieldLabel(field)}
          fullWidth
          disabled={readOnly}
          value={rhf.value == null || rhf.value === "" ? "" : String(rhf.value)}
          onChange={(event) => {
            const raw = event.target.value;
            // Empty clears the FK (null); otherwise store the related PK type.
            rhf.onChange(raw === "" ? null : toRelId(raw));
          }}
          error={!!fieldState.error || !!loadError}
          helperText={fieldState.error?.message ?? (loadError || helper)}
          slotProps={{ select: { displayEmpty: true } }}
        >
          <MenuItem value="">
            <em>{field.required ? "Select…" : "None"}</em>
          </MenuItem>
          {options.map((option) => (
            <MenuItem key={option.value} value={option.value}>
              {option.label}
            </MenuItem>
          ))}
        </TextField>
      )}
    />
  );
}

function fieldLabel(field: FieldMeta): string {
  if (field.type === "relation" && field.related) {
    if (field.related.kind === "belongs_to") {
      return `${field.name} (${field.related.slug})`;
    }
    return `${field.name} (${field.related.kind})`;
  }
  return field.name;
}

function inputTypeFor(field: FieldMeta): string {
  switch (field.type) {
    case "integer":
    case "float":
      return "number";
    // decimal stays a text input: it is submitted as an exact string, and a
    // number input would coerce it to a float.
    case "datetime":
      return "datetime-local";
    case "date":
      return "date";
    default:
      return "text";
  }
}

function helperText(field: FieldMeta): string | undefined {
  if (field.type === "json") {
    return "JSON object or array";
  }
  if (field.type === "relation" && field.related?.kind === "belongs_to") {
    return `Foreign key for ${field.related.slug}`;
  }
  if (isHasMany(field)) {
    return "has_many is meta-only";
  }
  return undefined;
}
