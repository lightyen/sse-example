import axios from "axios"
import { Task } from "redux-saga"
import { cancel, delay, fork, put, takeEvery } from "redux-saga/effects"
import * as ac from "./action"

export type AlertType = "warning" | "error" | "info" | "success"

export interface AlertOption {
	type: AlertType
	id?: string // intl id
	text?: string // raw message
}

export interface WeakAlertOption {
	type?: AlertType
	/** intl id */
	id?: string
	/** raw message */
	text?: string
	error?: unknown
}

export interface ErrResponse {
	code: number
	msg: string
}

export interface AnyResponse {
	data?: unknown
	error?: ErrResponse
}

export function nomalizeAlertOption({ type = "error", text, id, error }: WeakAlertOption): AlertOption {
	if (!(error instanceof Error)) {
		if (typeof id === "string" || typeof text === "string") {
			return { type, id, text }
		}
		return { type: "error", text: "Unknown" }
	}

	if (axios.isAxiosError<AnyResponse>(error)) {
		if (error.response?.data?.error?.msg) {
			return { type, id, text: error.response.data.error.msg }
		}
		return { type, id, text: error.message }
	}

	return { type, id, text: error.message }
}

function* closeAlertAfter() {
	yield delay(5000)
	yield put(ac.closeAlert())
}

export function autoCloseAlert() {
	return function* () {
		let task: Task
		yield takeEvery([ac.newAlert, ac.closeAlert], function* (action) {
			if (task && task.isRunning()) {
				yield cancel(task)
			}

			if (action.type === ac.closeAlert.type) {
				return
			}

			task = yield fork(closeAlertAfter)
		})
	}
}

/** alert saga */
export function* alert(option: WeakAlertOption) {
	yield put(ac.closeAlert())
	yield delay(25)
	yield put(ac.newAlert(option))
}
