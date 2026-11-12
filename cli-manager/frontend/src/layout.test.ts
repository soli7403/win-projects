import { describe, expect, it } from 'vitest'
import { panes, splitPane, removePane, swapPanes, movePane, setPaneSession, isEmpty, type PaneNode, type LayoutNode } from './layout'

const P = (id: string, sessionId: string): PaneNode => ({ type: 'pane', id, sessionId })
const ids = (n: LayoutNode): string[] => panes(n).map((p) => p.id)

describe('3.8 칸 관리', () => {
  it('빈 칸으로 만들어도 배치가 유지되고, 빈 칸을 다시 제거하면 구조가 접힌다', () => {
    const root = splitPane(P('a', 's1'), 'a', 'row', P('b', 's2'))
    const emptied = setPaneSession(root, 'b', '')
    expect(ids(emptied)).toEqual(['a', 'b'])
    expect(isEmpty(panes(emptied)[1])).toBe(true)
    const removed = removePane(emptied, 'b')!
    expect(removed.type).toBe('pane')
  })

  it('가운데 놓기: 두 칸의 세션을 맞바꾼다', () => {
    const root = splitPane(P('a', 's1'), 'a', 'row', P('b', 's2'))
    const swapped = swapPanes(root, 'a', 'b')
    expect(panes(swapped).map((p) => p.sessionId)).toEqual(['s2', 's1'])
    expect(ids(swapped)).toEqual(['a', 'b'])
  })

  it('가장자리에 놓기: 그 방향으로 칸을 옮긴다', () => {
    let root: LayoutNode = splitPane(P('a', 's1'), 'a', 'row', P('b', 's2'))
    root = splitPane(root, 'b', 'row', P('c', 's3')) // a | b | c
    const moved = movePane(root, 'c', 'a', 'left') // c | a | b
    expect(ids(moved)).toEqual(['c', 'a', 'b'])
    const down = movePane(root, 'c', 'a', 'down') // a 아래에 c
    expect(ids(down)).toEqual(['a', 'c', 'b'])
    const top = down.type === 'split' ? down.children[0] : down
    expect(top.type === 'split' && top.dir === 'col').toBe(true)
  })

  it('splitPane before 옵션은 앞에 끼운다', () => {
    const root = splitPane(P('a', 's1'), 'a', 'col', P('b', 's2'), true)
    expect(ids(root)).toEqual(['b', 'a'])
  })
})
