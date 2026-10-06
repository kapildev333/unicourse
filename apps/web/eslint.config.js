import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import jsxA11y from 'eslint-plugin-jsx-a11y'

export default tseslint.config(
  { ignores: ['dist', 'src/routeTree.gen.ts', 'src/api/types.ts'] },
  {
    files: ['**/*.{ts,tsx}'],
    extends: [...tseslint.configs.recommended, jsxA11y.flatConfigs.recommended],
    plugins: { 'react-hooks': reactHooks },
    rules: reactHooks.configs.recommended.rules,
  },
)
