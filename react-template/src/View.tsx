import { useAction } from "~/store"
import { Terminal } from "./Terminal"

export default function Content() {
	const { command, cancelCommand } = useAction().terminal
	return (
		<div style={{ padding: "2rem", background: "#787878" }}>
			<button
				onClick={() => {
					command("ping 8.8.8.8")
				}}
			>
				Submit
			</button>
			<button
				onClick={() => {
					cancelCommand()
				}}
			>
				Cancel
			</button>
			<Terminal />
		</div>
	)
}
