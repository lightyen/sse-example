import { eventChannel } from "redux-saga"
import { put, select, take, takeEvery } from "redux-saga/effects"
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
		foreground: "#adecff",
		background: "#222222",
		cursor: "#eeeeee",
		cursorAccent: "#222222",
		black: "#222222",
	},
})

const printable_ascii = /^[\x20-\x7E]$/

term.loadAddon(new FitAddon())
// for (let i = 0xa0; i <= 0xd6; i++) {
// 	term.write(String.fromCharCode(0xe000 + i) + " ")
// }

export interface KeyEvent {
	key: string
	domEvent: KeyboardEvent
}

let buffer = ""
let count = 0

function key(terminal: Terminal) {
	return eventChannel(emit => {
		const d = terminal.onKey(e => emit(e))
		return () => d.dispose()
	})
}

export default function* saga() {
	yield takeEvery(ac.eCommand, function* ({ payload }) {
		yield term.write(payload)
	})
	yield takeEvery(ac.eCommandEOF, function* ({ payload }) {
		yield term.write(payload)
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
					term.write("\x1b[D\x1b[K")
					count--
					buffer = buffer.slice(0, -1)
				}
				continue
			}

			console.log(JSON.stringify({ key }))
			const running = yield select(state => state.terminal.running)

			switch (key) {
				case "\x03":
					yield put(ac.cancelCommand())
					break
				case "\f":
					term.clear()
					break
				case "\r":
					if (running) continue
					term.write("\r\n")
					if (!buffer) {
						yield put(ac.command(""))
					} else {
						console.log(buffer)
						yield put(ac.command(buffer))
					}
					buffer = ""
					count = 0
					break
				case "\x1b":
					term.blur()
					break
				default:
					if (printable_ascii.test(key)) {
						term.write(key)
						count++
						buffer += key
					}
			}
		}
	})
}
