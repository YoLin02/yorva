import { describe, expect, it } from "vitest";
import { APP_NAME } from "./appMetadata";

describe("desktop metadata", () => {
  it("uses the product name", () => {
    expect(APP_NAME).toBe("YORVA");
  });
});
