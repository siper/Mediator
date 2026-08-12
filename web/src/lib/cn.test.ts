import { cn } from "@/lib/cn";

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("px-2", "px-4")).toBe("px-4");
  });

  it("handles conditional classes", () => {
    expect(cn("text-sm", false && "text-lg", "font-bold")).toBe("text-sm font-bold");
  });

  it("handles undefined values", () => {
    expect(cn("a", undefined, "b")).toBe("a b");
  });

  it("handles empty input", () => {
    expect(cn()).toBe("");
  });
});
