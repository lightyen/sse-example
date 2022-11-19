import yaml from "@rollup/plugin-yaml"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"
import eslint from "vite-plugin-eslint"
import svg from "vite-plugin-svgr"
import tsConfigPaths from "vite-plugin-tsconfig-paths"

export default defineConfig({
	plugins: [
		svg({ exportAsDefault: true }),
		yaml(),
		eslint(),
		tsConfigPaths(),
		react(),
	],
	esbuild: {
		logOverride: { "this-is-undefined-in-esm": "silent" },
	},
})
