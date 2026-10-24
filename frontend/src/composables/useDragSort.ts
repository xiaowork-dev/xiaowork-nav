import { ref, type Ref } from 'vue'

/**
 * Minimal native HTML5 drag-and-drop reordering for a list.
 *
 * Desktop uses drag-and-drop; pages should also expose up/down buttons for
 * touch devices (see the admin pages).
 */
export function useDragSort<T>(
  items: Ref<T[]>,
  getId: (item: T) => number,
  onCommit: (ordered: T[]) => void | Promise<void>,
) {
  const draggingId = ref<number | null>(null)
  const overId = ref<number | null>(null)

  function onDragStart(item: T): void {
    draggingId.value = getId(item)
  }

  function onDragEnd(): void {
    draggingId.value = null
    overId.value = null
  }

  function onDragOver(item: T, event: DragEvent): void {
    event.preventDefault()
    overId.value = getId(item)
  }

  function onDrop(target: T): void {
    const fromId = draggingId.value
    const toId = getId(target)
    draggingId.value = null
    overId.value = null
    if (fromId === null || fromId === toId) return

    const list = [...items.value]
    const fromIndex = list.findIndex((i) => getId(i) === fromId)
    const toIndex = list.findIndex((i) => getId(i) === toId)
    if (fromIndex < 0 || toIndex < 0) return

    const [moved] = list.splice(fromIndex, 1)
    list.splice(toIndex, 0, moved)
    items.value = list
    void onCommit(list)
  }

  return { draggingId, overId, onDragStart, onDragEnd, onDragOver, onDrop }
}
