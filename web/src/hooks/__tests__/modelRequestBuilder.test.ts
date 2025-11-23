import { describe, it, expect } from 'vitest'

/**
 * Issue #87/#88 回归测试
 * 验证同一 provider 的多个 model 在构建请求时不会互相覆盖
 */

// 模拟 AIModel 类型
interface AIModel {
  id: string
  provider: string
  name: string
  enabled: boolean
  apiKey?: string
  customApiUrl?: string
  customModelName?: string
}

// 提取 useTraderActions.ts 中的 buildRequest 逻辑进行测试
// 这个函数模拟了实际的请求构建逻辑
function buildModelRequest(models: AIModel[]) {
  return {
    models: Object.fromEntries(
      models.map((model) => [
        model.id, // Issue #87: 使用 id 而不是 provider
        {
          enabled: model.enabled,
          api_key: model.apiKey || '',
          custom_api_url: model.customApiUrl || '',
          custom_model_name: model.customModelName || '',
        },
      ])
    ),
  }
}

// 旧版有 bug 的实现（使用 provider 作为 key）
function buildModelRequestOldBuggy(models: AIModel[]) {
  return {
    models: Object.fromEntries(
      models.map((model) => [
        model.provider, // BUG: 同一 provider 的多个 model 会互相覆盖
        {
          enabled: model.enabled,
          api_key: model.apiKey || '',
          custom_api_url: model.customApiUrl || '',
          custom_model_name: model.customModelName || '',
        },
      ])
    ),
  }
}

describe('Issue #87/#88: Multiple AI models per provider', () => {
  const modelsWithSameProvider: AIModel[] = [
    {
      id: 'deepseek-chat',
      provider: 'deepseek',
      name: 'DeepSeek Chat',
      enabled: true,
      apiKey: 'sk-key-1',
      customModelName: 'deepseek-chat',
    },
    {
      id: 'deepseek-reasoner',
      provider: 'deepseek',
      name: 'DeepSeek Reasoner',
      enabled: true,
      apiKey: 'sk-key-2',
      customModelName: 'deepseek-reasoner',
    },
  ]

  describe('buildModelRequest (fixed version using model.id)', () => {
    it('should preserve all models with the same provider', () => {
      const request = buildModelRequest(modelsWithSameProvider)

      // 验证两个 model 都存在
      expect(Object.keys(request.models)).toHaveLength(2)
      expect(request.models['deepseek-chat']).toBeDefined()
      expect(request.models['deepseek-reasoner']).toBeDefined()
    })

    it('should use model.id as key, not provider', () => {
      const request = buildModelRequest(modelsWithSameProvider)

      // 验证 key 是 id 而不是 provider
      expect(request.models['deepseek-chat'].custom_model_name).toBe('deepseek-chat')
      expect(request.models['deepseek-reasoner'].custom_model_name).toBe('deepseek-reasoner')
    })

    it('should preserve individual API keys for each model', () => {
      const request = buildModelRequest(modelsWithSameProvider)

      expect(request.models['deepseek-chat'].api_key).toBe('sk-key-1')
      expect(request.models['deepseek-reasoner'].api_key).toBe('sk-key-2')
    })
  })

  describe('buildModelRequestOldBuggy (demonstrates the bug)', () => {
    it('should FAIL: later model overwrites earlier one with same provider', () => {
      const request = buildModelRequestOldBuggy(modelsWithSameProvider)

      // 旧版 bug: 只有一个 model 保留，另一个被覆盖
      expect(Object.keys(request.models)).toHaveLength(1) // 只有 1 个！
      expect(request.models['deepseek']).toBeDefined()

      // 第二个 model 覆盖了第一个
      expect(request.models['deepseek'].custom_model_name).toBe('deepseek-reasoner')
      expect(request.models['deepseek'].api_key).toBe('sk-key-2')
    })
  })

  describe('mixed providers', () => {
    const mixedModels: AIModel[] = [
      { id: 'openai-1', provider: 'openai', name: 'GPT-5.1', enabled: true, apiKey: 'sk-openai' },
      { id: 'deepseek-1', provider: 'deepseek', name: 'DeepSeek', enabled: true, apiKey: 'sk-ds-1' },
      { id: 'deepseek-2', provider: 'deepseek', name: 'DeepSeek Pro', enabled: false, apiKey: 'sk-ds-2' },
    ]

    it('should correctly handle multiple providers with multiple models each', () => {
      const request = buildModelRequest(mixedModels)

      expect(Object.keys(request.models)).toHaveLength(3)
      expect(request.models['openai-1'].api_key).toBe('sk-openai')
      expect(request.models['deepseek-1'].api_key).toBe('sk-ds-1')
      expect(request.models['deepseek-2'].api_key).toBe('sk-ds-2')
      expect(request.models['deepseek-2'].enabled).toBe(false)
    })
  })

  /**
   * 真实端到端场景测试
   * 模拟：用户已有一个 OpenAI 配置，再从模板添加第二个
   */
  describe('Real E2E scenario: adding second model from template', () => {
    // 模拟 supportedModels（模板）
    const supportedModels: AIModel[] = [
      { id: 'openai', provider: 'openai', name: 'OpenAI', enabled: false },
      { id: 'deepseek', provider: 'deepseek', name: 'DeepSeek', enabled: false },
    ]

    // 模拟 allModels（用户已配置，ID 是后端生成的唯一 ID）
    const existingUserModels: AIModel[] = [
      {
        id: 'user_openai_1763868235524084438', // 后端生成的唯一 ID
        provider: 'openai',
        name: 'OpenAI',
        enabled: true,
        apiKey: 'sk-first-key'
      },
    ]

    // 模拟 handleSaveModel 的逻辑
    function simulateAddNewModel(
      allModels: AIModel[],
      supportedModels: AIModel[],
      selectedTemplateId: string,
      newApiKey: string
    ): AIModel[] {
      const existingModel = allModels.find((m) => m.id === selectedTemplateId)
      const modelToUpdate = existingModel || supportedModels.find((m) => m.id === selectedTemplateId)

      if (!modelToUpdate) throw new Error('Model not found')

      if (existingModel) {
        // 更新现有
        return allModels.map((m) =>
          m.id === selectedTemplateId ? { ...m, apiKey: newApiKey } : m
        )
      } else {
        // 添加新配置（使用模板的 id）
        const newModel = { ...modelToUpdate, apiKey: newApiKey, enabled: true }
        return [...allModels, newModel]
      }
    }

    it('should add second model without overwriting first (template id != existing id)', () => {
      // 用户选择 "openai" 模板添加第二个 OpenAI 配置
      const updatedModels = simulateAddNewModel(
        existingUserModels,
        supportedModels,
        'openai', // 选择模板
        'sk-second-key'
      )

      // 应该有两个模型
      expect(updatedModels).toHaveLength(2)

      // 第一个是已有的（保持不变）
      expect(updatedModels[0].id).toBe('user_openai_1763868235524084438')
      expect(updatedModels[0].apiKey).toBe('sk-first-key')

      // 第二个是新添加的（使用模板 id "openai"）
      expect(updatedModels[1].id).toBe('openai')
      expect(updatedModels[1].apiKey).toBe('sk-second-key')
    })

    it('request should contain both models with different keys', () => {
      const updatedModels = simulateAddNewModel(
        existingUserModels,
        supportedModels,
        'openai',
        'sk-second-key'
      )

      const request = buildModelRequest(updatedModels)

      // 验证 request 包含两个不同的 key
      expect(Object.keys(request.models)).toHaveLength(2)
      expect(request.models['user_openai_1763868235524084438']).toBeDefined()
      expect(request.models['openai']).toBeDefined()

      // 两个 key 对应不同的 API key
      expect(request.models['user_openai_1763868235524084438'].api_key).toBe('sk-first-key')
      expect(request.models['openai'].api_key).toBe('sk-second-key')
    })

    it('OLD BUGGY version would fail: both models would use same provider key', () => {
      const updatedModels = simulateAddNewModel(
        existingUserModels,
        supportedModels,
        'openai',
        'sk-second-key'
      )

      const request = buildModelRequestOldBuggy(updatedModels)

      // 旧版 bug：两个模型都用 provider 作为 key，会覆盖！
      expect(Object.keys(request.models)).toHaveLength(1) // 只有 1 个！
      expect(request.models['openai'].api_key).toBe('sk-second-key') // 第二个覆盖了第一个
    })
  })
})
