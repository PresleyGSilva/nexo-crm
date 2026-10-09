/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: "class",
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        brand: {
          50: "#eef4ff",
          100: "#dbe7fe",
          500: "#2f6bff",
          600: "#2456d6",
          700: "#1d45ad",
        },
      },
    },
  },
  plugins: [],
};
