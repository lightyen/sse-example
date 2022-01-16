import ReactRefreshPlugin from "@pmmmwh/react-refresh-webpack-plugin"
import ForkTsCheckerPlugin from "fork-ts-checker-webpack-plugin"
import path from "path"
import { merge } from "webpack-merge"
import createBaseConfig from "./webpack.common"

process.env.NODE_ENV = "development"
process.env.PUBLIC_URL = ""

export default merge(createBaseConfig(), {
	mode: "development",
	devtool: "inline-source-map",
	// @ts-expect-error
	devServer: {
		hot: true,
		compress: false,
		open: true,
		historyApiFallback: true,
		proxy: [
			{
				context: ["/stream"],
				target: "http://localhost:8080",
			},
		],
	},
	stats: {
		children: false,
		modules: false,
		entrypoints: false,
	},
	performance: {
		hints: false,
		assetFilter: (filename: string) => {
			return filename.endsWith(".css") || filename.endsWith(".js")
		},
	},
	cache: {
		type: "memory",
	},
	plugins: [
		new ForkTsCheckerPlugin({
			typescript: {
				configFile: path.resolve(__dirname, "../src/tsconfig.json"),
			},
			logger: {
				devServer: false,
			},
		}),
		new ReactRefreshPlugin(),
	],
})
