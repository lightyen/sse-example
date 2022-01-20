import type { ActionCreatorWithoutPayload } from "@reduxjs/toolkit"
import axios from "axios"
import type { Task } from "redux-saga"
import { eventChannel } from "redux-saga"
import { call, cancel, cancelled, fork, put, select, take, takeEvery } from "redux-saga/effects"
import { eTerminal, eTerminalEOF, eClearTerminal } from "../terminal/action"
import * as ac from "./action"

function event(source: EventSource, event: string) {
	return eventChannel(emit => {
		source.addEventListener(event, e => emit(e))
		return () => void 0
	})
}

function handleEventStream(url: string) {
	return fork(function* () {
		const source = new EventSource(url)

		yield fork(function* () {
			const ch = event(source, "error")
			while (true) {
				yield take(ch)
				yield put(ac.errorEventStream())
			}
		})

		let tasks: Task[] = []

		try {
			tasks = [
				yield fork(function* () {
					const ch = event(source, "terminal")
						while (true) {
							const { data } = yield take(ch)
							const resp = JSON.parse(data)
							switch (resp.type) {
								case "eof":
									yield put(eTerminalEOF(resp.data))
									break
								case "out":
									yield put(eTerminal(resp.data))
									break
								case "clear":
									yield put(eClearTerminal())
									break
							}
						}
				}),
				yield fork(function* () {
					const ch = event(source, "timecount")
					while (true) {
						const { data } = yield take(ch)
						yield put(ac.eCount(data))
					}
				}),
			]

			const ch = event(source, "establish")
			while (true) {
				const { lastEventId } = yield take(ch)
				axios.defaults.headers.common["Last-Event-ID"] = lastEventId
				yield put(ac.establishEventStream(source))
			}
		} finally {
			if (cancelled()) {
				yield cancel(tasks)
				source.close()
			}
		}
	})
}

function eventStream(openAC: ActionCreatorWithoutPayload) {
	let task: Task | undefined
	return function* (action: ReturnType<ActionCreatorWithoutPayload>) {
		yield clear()
		if (action.type === openAC.type) {
			task = yield handleEventStream("/stream")
		}
	}
	function* clear() {
		if (task) yield cancel(task)
		task = undefined
	}
}

function* getSource() {
	const source: EventSource | undefined = yield select(state => state.app.source)
	return source
}

export default function* saga() {
	yield takeEvery([ac.openEventStream, ac.closeEventStream], eventStream(ac.openEventStream))

	yield takeEvery(ac.timecount, function* (a) {
		try {
			if (a.payload === false) {
				yield call(axios.get, `/stream/timecount`, { params: { enable: "off" } })
			} else {
				const source: EventSource | undefined = yield getSource()
				if (!source) {
					yield take(ac.establishEventStream)
				}
				yield call(axios.get, `/stream/timecount`)
			}
		} catch {}
	})

	yield put(ac.openEventStream())
}
