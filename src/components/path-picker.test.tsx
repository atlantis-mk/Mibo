import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import { PathPicker } from './path-picker'

describe('PathPicker', () => {
  it('does not auto-browse again when only the browse callback identity changes', async () => {
    const browse = vi.fn().mockResolvedValue({
      provider: 'openlist',
      root_path: '/',
      current_path: '/',
      items: [],
    })
    const onValueChange = vi.fn()

    function Harness() {
      const [, setRenderCount] = useState(0)

      return (
        <>
          <button
            type='button'
            onClick={() => setRenderCount((count) => count + 1)}
          >
            Rerender
          </button>
          <PathPicker
            browse={(path, options) => browse(path, options)}
            browseKey='openlist:http://127.0.0.1:5244:'
            browseLabel='当前浏览目录'
            value='/'
            placeholder='/'
            onValueChange={onValueChange}
            selectCurrentOnBrowse
            ready
          />
        </>
      )
    }

    const { getByRole } = await render(<Harness />)

    await vi.waitFor(() => expect(browse).toHaveBeenCalledOnce())
    await userEvent.click(getByRole('button', { name: 'Rerender' }))

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(browse).toHaveBeenCalledOnce()
    expect(onValueChange).not.toHaveBeenCalled()
  })
})
