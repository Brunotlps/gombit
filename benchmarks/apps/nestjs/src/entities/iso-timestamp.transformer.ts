import { ValueTransformer } from 'typeorm';

// Makes "a timestamptz column is an ISO-8601 string with microseconds" an
// entity-level fact, rather than string surgery the service happens to do on
// a value it assumes is always a raw pg string with a +00 offset.
//
// The pg driver is configured (data-source.ts) to return timestamptz as a raw
// string like "2026-08-26 16:20:21.962946+00", preserving the microseconds a
// JS Date cannot hold, and the session TZ is forced to UTC so the offset is
// always +00. This transformer's `from` reshapes that to the canonical
// "2026-08-26T16:20:21.962946Z" the siblings emit. It is defensive on two
// axes the review flagged:
//   - if it ever receives a JS Date (e.g. a future TypeORM that hydrates the
//     `timestamptz` alias, or the pg parser override being bypassed), it
//     returns a valid ISO string instead of throwing on `.replace` — the
//     microsecond read-path test then catches the precision loss rather than
//     the app 500ing;
//   - if the offset is somehow not +00 (it can't be, with UTC forced), it
//     falls back to a correct UTC conversion instead of emitting a bare +00.
export const isoTimestamp: ValueTransformer = {
  to: (value: string): string => value,
  from: (value: string | Date | null): string | null => {
    if (value === null || value === undefined) {
      return value as null;
    }
    if (value instanceof Date) {
      return value.toISOString();
    }
    const withT = value.replace(' ', 'T');
    if (/[+-]00(:?00)?$/.test(withT)) {
      return withT.replace(/[+-]00(:?00)?$/, 'Z');
    }
    return new Date(withT).toISOString();
  },
};
