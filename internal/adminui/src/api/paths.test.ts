import { describe, expect, it } from "vitest";

import { apiResourcePath, pathSegment, spaDetailPath, spaEditPath } from "./paths";

describe("pathSegment", () => {
  it("encodes slash and parent-directory PKs as one segment", () => {
    expect(pathSegment("foo/bar")).toBe("foo%2Fbar");
    expect(pathSegment("../widgets/1")).toBe("..%2Fwidgets%2F1");
  });

  it("leaves ordinary integer and UUID PKs unchanged", () => {
    expect(pathSegment("42")).toBe("42");
    expect(pathSegment("550e8400-e29b-41d4-a716-446655440000")).toBe(
      "550e8400-e29b-41d4-a716-446655440000",
    );
  });
});

describe("apiResourcePath", () => {
  it("does not let new URL() collapse a parent-directory PK onto another resource", () => {
    const path = apiResourcePath("items", "../widgets/1");
    expect(path).toBe("/admin/resources/items/..%2Fwidgets%2F1");
    const url = new URL("http://127.0.0.1" + path);
    expect(url.pathname).toBe("/admin/resources/items/..%2Fwidgets%2F1");
    expect(url.pathname).not.toBe("/admin/resources/widgets/1");
  });

  it("keeps foo/bar as one segment", () => {
    const path = apiResourcePath("items", "foo/bar");
    expect(path).toBe("/admin/resources/items/foo%2Fbar");
    expect(new URL("http://127.0.0.1" + path).pathname).toBe("/admin/resources/items/foo%2Fbar");
  });
});

describe("spa paths", () => {
  it("encodes the id in detail and edit routes", () => {
    expect(spaDetailPath("items", "foo/bar")).toBe("/items/foo%2Fbar");
    expect(spaEditPath("items", "../widgets/1")).toBe("/items/..%2Fwidgets%2F1/edit");
  });
});
