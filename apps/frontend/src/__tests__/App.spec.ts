import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import App from '../App.vue'

describe('App', () => {
  it('renders the router outlet', () => {
    const wrapper = mount(App, {
      global: {
        stubs: {
          RouterView: {
            template: '<div data-testid="router-view-stub">Router outlet</div>',
          },
        },
      },
    })

    expect(wrapper.find('[data-testid="router-view-stub"]').exists()).toBe(true)
  })
})
