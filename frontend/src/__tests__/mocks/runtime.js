const listeners = {}

export function EventsOn(event, handler) {
  if (!listeners[event]) listeners[event] = []
  listeners[event].push(handler)
}

export function EventsOff(event) {
  delete listeners[event]
}

export function EventsEmit(event, ...args) {
  if (listeners[event]) {
    listeners[event].forEach(h => h(...args))
  }
}

export function clearEventMocks() {
  for (const key in listeners) delete listeners[key]
}
