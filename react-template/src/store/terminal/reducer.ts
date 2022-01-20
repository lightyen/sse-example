import { createReducer } from "@reduxjs/toolkit"
export interface TerminalStore {}

const init: TerminalStore = {}

export const terminal = createReducer(init, builder => builder)
