import { describe, expect, it } from "vitest";

import formSrc from "./ResourceFormPage.tsx?raw";

describe("ResourceFormPage", () => {
  it("does not GET or PATCH an edit form unless the row can be hydrated", () => {
    expect(formSrc).toMatch(/canPopulateEditForm\(model\)/);
    expect(formSrc).toMatch(/if \(mode === "edit" && !rowLoaded\) \{/);
    expect(formSrc).toMatch(/reset\(rowToFormValues\(envelope\.data, model\.fields\)\);\s*setRowLoaded\(true\);/s);
  });
});
