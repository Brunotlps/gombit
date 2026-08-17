import type { FieldValues, Path, UseFormSetError } from "react-hook-form";

import { ContractError, isD10ErrorBody } from "./generated/client";

/**
 * Map a D10 `error.fields` payload onto React Hook Form field errors.
 * Accepts `ContractError` or a D10 error body. Returns true when at least
 * one field error was set.
 */
export function applyContractErrors<TFieldValues extends FieldValues>(
  setError: UseFormSetError<TFieldValues>,
  err: unknown,
): boolean {
  const fields = d10Fields(err);
  if (fields === undefined) {
    return false;
  }
  let applied = false;
  for (const [name, messages] of Object.entries(fields)) {
    if (name === "") {
      continue;
    }
    const message = (messages ?? []).filter((item) => item.trim() !== "").join("; ");
    if (message === "") {
      continue;
    }
    setError(name as Path<TFieldValues>, { type: "server", message });
    applied = true;
  }
  return applied;
}

function d10Fields(err: unknown): { [key: string]: string[] } | undefined {
  if (err instanceof ContractError) {
    return err.fields;
  }
  if (isD10ErrorBody(err)) {
    return err.error.fields;
  }
  return undefined;
}
