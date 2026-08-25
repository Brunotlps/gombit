import { describe, expect, it } from "vitest";

import routerSrc from "./router.tsx?raw";

describe("admin list route", () => {
  it("remounts ResourceListPage when the slug param changes", () => {
    expect(routerSrc).toMatch(/<ResourceListPage\s+key=\{slug\}\s*\/>/);
    expect(routerSrc).toMatch(/<Route\s+path=":slug"\s+element=\{<ResourceListRoute\s*\/>\}\s*\/>/);
  });
});

describe("admin form route", () => {
  it("remounts ResourceFormPage when the slug or id param changes", () => {
    expect(routerSrc).toMatch(
      /<ResourceFormPage\s+key=\{`\$\{slug\}-\$\{id \|\| "new"\}`\}\s+mode=\{mode\}\s*\/>/,
    );
    expect(routerSrc).toMatch(
      /<Route\s+path=":slug\/new"\s+element=\{<ResourceFormRoute\s+mode="create"\s*\/>\}\s*\/>/,
    );
    expect(routerSrc).toMatch(
      /<Route\s+path=":slug\/:id\/edit"\s+element=\{<ResourceFormRoute\s+mode="edit"\s*\/>\}\s*\/>/,
    );
  });
});
