const handlers = {}

export function mockWailsBinding(name, fn) {
  handlers[name] = fn
}

export function clearWailsMocks() {
  for (const key in handlers) delete handlers[key]
}

export function __getHandler(name) {
  return handlers[name] || (() => Promise.reject(new Error(`Not mocked: ${name}`)))
}
