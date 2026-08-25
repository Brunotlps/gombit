import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

const routerSrc = readFileSync(join(dirname(fileURLToPath(import.meta.url)), "router.tsx"), "utf8");

describe("admin list route", () => {
  it("remounts ResourceListPage when the slug param changes", () => {
    expect(routerSrc).toMatch(/<ResourceListPage\s+key=\{slug\}\s*\/>/);
    expect(routerSrc).toMatch(/<Route\s+path=":slug"\s+element=\{<ResourceListRoute\s*\/>\}\s*\/>/);
  });
});
