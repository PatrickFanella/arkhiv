import { renderToString } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
vi.mock("./Reader", () => ({ default: () => null }));
import App from "./App";
describe("catalog before the API responds", () => {
  it("shows loading without claiming an empty or imported catalog", () => {
    vi.stubGlobal("location", {
      search: "",
      pathname: "/",
      href: "http://localhost/",
    });
    const html = renderToString(<App />);
    expect(html).toContain("Loading the archive");
    expect(html).not.toContain("No documents have been imported");
    expect(html).not.toContain("Download original PDF");
    vi.unstubAllGlobals();
  });
});
