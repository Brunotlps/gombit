import { describe, expect, it } from "vitest";

import routerSrc from "./router.tsx?raw";

describe("admin list route", () => {
  it("remounts ResourceListPage when the slug param changes", () => {
    expect(routerSrc).toMatch(/<ResourceListPage\s+key=\{slug\}\s*\/>/);
    expect(routerSrc).toMatch(/<Route\s+path=":slug"\s+element=\{<ResourceListRoute\s*\/>\}\s*\/>/);
  });
});
