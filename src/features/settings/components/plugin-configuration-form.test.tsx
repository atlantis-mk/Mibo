import { describe, expect, it, vi } from 'vitest'
import { render } from 'vitest-browser-react'
import { userEvent } from 'vitest/browser'
import {
  buildPluginConfigurationDefaults,
  PluginConfigurationForm,
} from './plugin-configuration-form'

describe('PluginConfigurationForm', () => {
  it('applies schema defaults and keeps existing values', () => {
    const next = buildPluginConfigurationDefaults(
      {
        fields: [
          {
            key: 'base_url',
            type: 'url',
            default: 'https://plugin.example.com',
          },
          {
            key: 'timeout',
            type: 'duration',
            default: '10s',
          },
        ],
      },
      {
        timeout: '30s',
      }
    )

    expect(next).toEqual({
      base_url: 'https://plugin.example.com',
      timeout: '30s',
    })
  })

  it('renders redacted secrets without exposing the stored value', async () => {
    const handleChange = vi.fn()
    const { getByText, container } = await render(
      <PluginConfigurationForm
        schema={{
          fields: [
            {
              key: 'api_key',
              type: 'secret',
              required: true,
              display: {
                label: 'API 密钥',
              },
            },
          ],
        }}
        value={{ api_key: '***REDACTED***' }}
        onChange={handleChange}
      />
    )

    const input = container.querySelector('input[type="password"]')
    if (!(input instanceof HTMLInputElement)) {
      throw new Error('expected password input')
    }

    expect(input.value).toBe('')
    expect(input.placeholder).toBe('输入新值以替换')
    await expect
      .element(getByText('已配置的密钥不会回显，留空即可保持现有值。'))
      .toBeInTheDocument()
  })

  it('updates primitive field values through the shared change handler', async () => {
    let currentValue: Record<string, unknown> = {
      enabled: false,
      timeout: '10s',
    }
    const handleChange = vi.fn((next: Record<string, unknown>) => {
      currentValue = next
    })
    const view = await render(
      <PluginConfigurationForm
        schema={{
          fields: [
            {
              key: 'timeout',
              type: 'duration',
              display: { label: '超时时间' },
            },
          ],
        }}
        value={currentValue}
        onChange={handleChange}
      />
    )

    const input = view.container.querySelector('input[type="text"]')
    if (!(input instanceof HTMLInputElement)) {
      throw new Error('expected text input')
    }

    await userEvent.fill(input, '45s')
    expect(handleChange).toHaveBeenCalled()
    expect(currentValue.timeout).toBe('45s')
  })
})
