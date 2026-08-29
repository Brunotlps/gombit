import { useEffect, useState } from "react";
import { Controller, type Control, type FieldValues } from "react-hook-form";

import { Checkbox, FormControlLabel, MenuItem, TextField } from "@mui/material";

import { useApiClient } from "../api/client";
import { apiResourcePath } from "../api/paths";
import type { FieldMeta, Row } from "../api/types";
import { isHasMany, isManyToMany } from "../fields";

type Props = {
  field: FieldMeta;
  control: Control<FieldValues>;
  disabled?: boolean;
};

type Option = { id: number; label: string };

export function FieldWidget({ field, control, disabled }: Props) {
  const readOnly = disabled || field.readonly || isHasMany(field);
  const label = fieldLabel(field);

  if (isManyToMany(field)) {
    return <RelationMultiSelect field={field} control={control} disabled={disabled} />;
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
 * RelationMultiSelect renders a many-to-many field as a multi-select backed by
 * the related model's admin list endpoint, showing the related label field and
 * submitting the selected ids (#223).
 */
function RelationMultiSelect({ field, control, disabled }: Props) {
  const client = useApiClient();
  const slug = field.related?.slug ?? "";
  const labelField = field.related?.label_field ?? "";
  const [options, setOptions] = useState<Option[]>([]);
  const [loadError, setLoadError] = useState("");

  useEffect(() => {
    if (!slug) {
      return;
    }
    let cancelled = false;
    void (async () => {
      try {
        const env = await client.get<Row[]>(apiResourcePath(slug), { per_page: 100 });
        if (cancelled) {
          return;
        }
        const rows = Array.isArray(env.data) ? env.data : [];
        setOptions(
          rows.map((r) => ({
            id: Number(r.id),
            label: labelField && r[labelField] != null ? String(r[labelField]) : `#${String(r.id)}`,
          })),
        );
      } catch (err: unknown) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : "failed to load options");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [client, slug, labelField]);

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
              rhf.onChange(list.map((v) => Number(v)));
            }}
            helperText={loadError || `Select related ${slug}`}
            error={!!loadError}
            slotProps={{ select: { multiple: true } }}
          >
            {options.map((option) => (
              <MenuItem key={option.id} value={String(option.id)}>
                {option.label}
              </MenuItem>
            ))}
          </TextField>
        );
      }}
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
    case "decimal":
      return "number";
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
