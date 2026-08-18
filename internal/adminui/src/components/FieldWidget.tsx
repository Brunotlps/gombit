import { Controller, type Control, type FieldValues } from "react-hook-form";

import { Checkbox, FormControlLabel, TextField } from "@mui/material";

import type { FieldMeta } from "../api/types";
import { isHasMany } from "../fields";

type Props = {
  field: FieldMeta;
  control: Control<FieldValues>;
  disabled?: boolean;
};

export function FieldWidget({ field, control, disabled }: Props) {
  const readOnly = disabled || field.readonly || isHasMany(field);
  const label = fieldLabel(field);

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
