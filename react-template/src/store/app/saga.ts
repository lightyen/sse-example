import type { ActionCreatorWithoutPayload } from "@reduxjs/toolkit"
import axios from "axios"
import type { Task } from "redux-saga"
import { eventChannel } from "redux-saga"
import { call, cancel, cancelled, fork, put, select, take, takeEvery } from "redux-saga/effects"
import { eShell, shell } from "../terminal/action"
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
					const ch = event(source, "shell")
					while (true) {
						const { data } = yield take(ch)
						const resp = JSON.parse(data)
						yield put(eShell(resp.data))
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
				yield put(shell(""))
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

	yield takeEvery(shell, function* ({ payload }) {
		try {
			yield call(axios.post, "/stream/shell", { input: payload })
		} catch {}
	})

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
