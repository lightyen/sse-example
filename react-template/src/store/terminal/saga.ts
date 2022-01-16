import { eventChannel } from "redux-saga"
import { put, take, takeEvery } from "redux-saga/effects"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import "xterm/css/xterm.css"
import "xterm/lib/xterm.js"
import * as ac from "./action"

export const term = new Terminal({
	fontFamily: "Cascadia Code PL",
	cursorBlink: true,
	allowProposedApi: true,
	allowTransparency: false,
	rendererType: "canvas",
	theme: {
		background: "#222222",
		cursor: "#eeeeee",
		cursorAccent: "#222222",
		black: "#222222",
	},
})

const printable_ascii = /^[\x20-\x7E]$/
let count = 0

term.loadAddon(new FitAddon())
// for (let i = 0xa0; i <= 0xd6; i++) {
// 	term.write(String.fromCharCode(0xe000 + i) + " ")
// }

export interface KeyEvent {
	key: string
	domEvent: KeyboardEvent
}

function key(terminal: Terminal) {
	return eventChannel(emit => {
		const d = terminal.onKey(e => emit(e))
		return () => d.dispose()
	})
}

export default function* saga() {
	yield takeEvery(ac.eShell, function* ({ payload }) {
		yield term.write(payload)
		if (payload === "\b") term.write("\x1b[P")
	})
	yield takeEvery(ac.openTerminal, function* () {
		const ch = key(term)
		while (true) {
			const e: KeyEvent = yield take(ch)
			const { domEvent, key } = e
			if (domEvent.key === "F5") {
				window.location.reload()
				continue
			}
			if (domEvent.key === "F12") {
				term.blur()
				continue
			}

			if (domEvent.key === "Backspace") {
				if (count > 0) {
					yield put(ac.shell("\b"))
					count--
				}
				continue
			}

			if (key === "\r") {
				count = 0
				term.write("\n")
			}

			console.log(JSON.stringify({ key }))

			if (printable_ascii.test(key)) {
				count++
			}

			yield put(ac.shell(key))
		}
	})
}
