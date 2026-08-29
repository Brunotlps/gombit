import { useEffect, useState } from "react";
import { Controller, type Control, type FieldValues } from "react-hook-form";

import { Autocomplete, Checkbox, FormControlLabel, TextField } from "@mui/material";

import { useApiClient } from "../api/client";
import { useCatalog } from "../app/providers";
import type { FieldMeta, Row } from "../api/types";
import {
  isBelongsTo,
  isHasMany,
  isManyToMany,
  type RelId,
  relationListQuery,
  relationOptions,
  toIdList,
  toRelId,
} from "../fields";

type Props = {
  field: FieldMeta;
  control: Control<FieldValues>;
  disabled?: boolean;
};

type Option = { value: string; label: string };

// relationPageSize is the related-model page fetched for the picker
// (contract.MaxPerPage). When the related model has a configured Search, typing
// narrows this page server-side, so rows beyond it are reachable; otherwise the
// widget filters the loaded page client-side and flags the truncation.
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

/** useDebounced returns value after it has been stable for delayMs. */
function useDebounced(value: string, delayMs: number): string {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs);
    return () => clearTimeout(timer);
  }, [value, delayMs]);
  return debounced;
}

/** normalizeIds pulls the selected primary keys off a picker's form value. */
function normalizeIds(value: unknown, multiple: boolean): RelId[] {
  if (multiple) {
    return Array.isArray(value) ? (value as RelId[]) : [];
  }
  return value == null || value === "" ? [] : [value as RelId];
}

/**
 * RelationAutocomplete is the searchable picker shared by belongs_to (single)
 * and many_to_many (multiple). It loads the related model's rows from the
 * data-plane list endpoint (client.list -> /api/v1/admin/...); when the related
 * model has a configured Search it searches server-side (so rows past the first
 * page are reachable) and leaves Autocomplete's local filter off; otherwise it
 * falls back to client-side filtering of the loaded page. Already-selected ids
 * outside the current page are fetched via client.detail so they show a label,
 * not a raw key (#223).
 */
function RelationAutocomplete({
  field,
  multiple,
  value,
  onChange,
  disabled,
  fieldState,
}: {
  field: FieldMeta;
  multiple: boolean;
  value: unknown;
  onChange: (next: unknown) => void;
  disabled?: boolean;
  fieldState?: { error?: { message?: string } };
}) {
  const client = useApiClient();
  const { bySlug } = useCatalog();
  const slug = field.related?.slug ?? "";
  const labelField = field.related?.label_field ?? "";
  const related = bySlug.get(slug);
  const pkField = related?.pk ?? "id";
  const searchable = (related?.search?.length ?? 0) > 0;

  const [input, setInput] = useState("");
  const search = useDebounced(input, 250);
  const [options, setOptions] = useState<Option[]>([]);
  const [selectedLabels, setSelectedLabels] = useState<Map<string, Option>>(new Map());
  const [truncated, setTruncated] = useState(false);
  const [loadError, setLoadError] = useState("");
  const [loading, setLoading] = useState(false);

  // Load a page of options — server-side search only when the model supports it.
  useEffect(() => {
    if (!slug) {
      return;
    }
    let cancelled = false;
    setLoading(true);
    void (async () => {
      try {
        const env = await client.list(slug, relationListQuery(searchable, search, relationPageSize));
        if (cancelled) {
          return;
        }
        const rows = Array.isArray(env.data) ? env.data : [];
        setTruncated((env.meta?.total ?? rows.length) > relationPageSize);
        setOptions(relationOptions(rows, pkField, labelField));
        setLoadError("");
      } catch (err: unknown) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : "failed to load options");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [client, slug, labelField, pkField, searchable, search]);

  const ids = normalizeIds(value, multiple);
  const optionByValue = new Map(options.map((option) => [option.value, option]));
  const idsKey = ids.map(String).join(",");

  // Fetch labels for selected ids that are not on the current page, so an
  // already-selected out-of-page row shows its label instead of its key.
  useEffect(() => {
    if (!slug) {
      return;
    }
    const missing = ids
      .map(String)
      .filter((v) => !optionByValue.has(v) && !selectedLabels.has(v));
    if (missing.length === 0) {
      return;
    }
    let cancelled = false;
    void (async () => {
      const fetched = new Map(selectedLabels);
      await Promise.all(
        missing.map(async (id) => {
          try {
            const env = await client.detail(slug, id);
            const row = env.data as Row | undefined;
            const label = row && labelField && row[labelField] != null ? String(row[labelField]) : id;
            fetched.set(id, { value: id, label });
          } catch {
            // Leave it to the key fallback below.
          }
        }),
      );
      if (!cancelled) {
        setSelectedLabels(fetched);
      }
    })();
    return () => {
      cancelled = true;
    };
    // idsKey and options track when the selected set or the page changes.
  }, [client, slug, labelField, idsKey, options]); // eslint-disable-line react-hooks/exhaustive-deps

  const resolve = (id: RelId): Option => {
    const v = String(id);
    return optionByValue.get(v) ?? selectedLabels.get(v) ?? { value: v, label: v };
  };

  const helper = loadError
    ? loadError
    : searchable
      ? "Type to search"
      : truncated
        ? `Showing first ${relationPageSize} ${slug} — refine the model's Search to reach the rest`
        : `Select related ${slug}`;

  const acValue = multiple ? ids.map(resolve) : ids.length > 0 ? resolve(ids[0]) : null;

  return (
    <Autocomplete
      multiple={multiple}
      options={options}
      value={acValue as never}
      getOptionLabel={(option) => option.label}
      isOptionEqualToValue={(option, selected) => option.value === selected.value}
      filterOptions={searchable ? (opts) => opts : undefined}
      filterSelectedOptions={multiple}
      disabled={disabled}
      loading={loading}
      onInputChange={(_event, term, reason) => {
        // Only a user keystroke (or clearing it) drives the search term; a
        // select/reset sets the input to the chosen label and must not refetch.
        if (reason === "input" || reason === "clear") {
          setInput(term);
        }
      }}
      onChange={(_event, selected) => {
        if (multiple) {
          onChange(toIdList((selected as Option[]).map((option) => option.value)));
        } else {
          const one = selected as Option | null;
          onChange(one ? toRelId(one.value) : null);
        }
      }}
      renderInput={(params) => (
        <TextField
          {...params}
          label={fieldLabel(field)}
          fullWidth
          required={!multiple && field.required && !disabled}
          error={!!fieldState?.error || !!loadError}
          helperText={fieldState?.error?.message ?? helper}
        />
      )}
    />
  );
}

/** RelationMultiSelect renders a many-to-many field as a searchable multi-select. */
function RelationMultiSelect({ field, control, disabled }: Props) {
  return (
    <Controller
      name={field.name}
      control={control}
      render={({ field: rhf }) => (
        <RelationAutocomplete
          field={field}
          multiple
          value={rhf.value}
          onChange={rhf.onChange}
          disabled={disabled || field.readonly}
        />
      )}
    />
  );
}

/** RelationSelect renders a belongs_to foreign key as a searchable single-select. */
function RelationSelect({ field, control, disabled }: Props) {
  const readOnly = disabled || field.readonly;
  return (
    <Controller
      name={field.name}
      control={control}
      rules={{ required: field.required && !readOnly }}
      render={({ field: rhf, fieldState }) => (
        <RelationAutocomplete
          field={field}
          multiple={false}
          value={rhf.value}
          onChange={rhf.onChange}
          disabled={readOnly}
          fieldState={fieldState}
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
