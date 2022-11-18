import yaml from "@rollup/plugin-yaml"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"
import eslint from "vite-plugin-eslint"
import svg from "vite-plugin-svgr"
import tsConfigPaths from "vite-plugin-tsconfig-paths"
import tailwindConfig from "./tailwind.config"

export default defineConfig({
	plugins: [
		svg({ exportAsDefault: true }),
		yaml(),
		eslint(),
		tsConfigPaths(),
		react({
			jsxImportSource: "@emotion/react",
			babel: {
				plugins: [["twobj", { tailwindConfig, throwError: true }], "@emotion"],
			},
		}),
	],
	esbuild: {
		logOverride: { "this-is-undefined-in-esm": "silent" },
	},
	server: {
		proxy: {
			"^/apis/.*": `http://127.0.0.1:8080`,
		},
	},
})
