import { WorkerPoolManager } from "@pierre/diffs/worker"

export type WorkerPoolStyle = "unified" | "split"

function createInlineWorker(): Worker {
  const workerCode = `self.onmessage = async () => { };`
  const blob = new Blob([workerCode], { type: "application/javascript" })
  const url = URL.createObjectURL(blob)
  return new Worker(url, { type: "module" })
}

function createPool(lineDiffType: "none" | "word-alt") {
  const pool = new WorkerPoolManager(
    {
      workerFactory: createInlineWorker,
      poolSize: 2,
    },
    {
      theme: "OpenCode",
      lineDiffType,
      preferredHighlighter: "shiki-wasm",
    },
  )

  pool.initialize()
  return pool
}

let unified: WorkerPoolManager | undefined
let split: WorkerPoolManager | undefined

export function getWorkerPool(style: WorkerPoolStyle | undefined): WorkerPoolManager | undefined {
  if (typeof window === "undefined") return

  if (style === "split") {
    if (!split) split = createPool("word-alt")
    return split
  }

  if (!unified) unified = createPool("none")
  return unified
}

export function getWorkerPools() {
  return {
    unified: getWorkerPool("unified"),
    split: getWorkerPool("split"),
  }
}
