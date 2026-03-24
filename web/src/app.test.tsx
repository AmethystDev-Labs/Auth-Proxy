import React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { App } from "@/app";

describe("App", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ authenticated: false }),
      }),
    );
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("renders the simplified black login card", async () => {
    const { container } = render(<App />);

    await waitFor(() => {
      expect(screen.getByText("Login to your account")).toBeTruthy();
    });

    expect(screen.getByText("Enter your username below to login to your account")).toBeTruthy();
    expect(screen.getByLabelText("Username")).toBeTruthy();
    expect(screen.getByLabelText("Password")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Login" })).toBeTruthy();
    expect(screen.queryByText("AuthProxy Gate")).toBeNull();
    expect(screen.queryByText("Unlock upstream")).toBeNull();
    expect(container.querySelector("main")?.className).toContain("bg-black");
  });
});
