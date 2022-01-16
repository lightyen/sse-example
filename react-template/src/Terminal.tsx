import { useEffect, useRef } from "react"
import { useAction, useSelect } from "~/store"

export function Terminal() {
	const { openTerminal, closeTerminal } = useAction().terminal
	const term = useSelect(state => state.terminal.term)
	const ref = useRef<HTMLDivElement>()
	useEffect(() => {
		openTerminal(ref.current)
		return () => {
			closeTerminal()
		}
	}, [openTerminal, closeTerminal, term])
	return (
		<div style={{ display: "flex", justifyContent: "center" }}>
			<div ref={ref} style={{ overflow: "hidden" }} />
		</div>
	)
}
