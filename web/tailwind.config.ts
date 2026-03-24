import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#0f172a",
        ember: "#d97706",
        parchment: "#fffaf0",
      },
      boxShadow: {
        flare: "0 24px 60px rgba(15, 23, 42, 0.18)",
      },
    },
  },
  plugins: [],
} satisfies Config;
