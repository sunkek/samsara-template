import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// Each test gets a fresh DOM; a leaked render makes the next test's queries
// ambiguous rather than failing outright.
afterEach(cleanup)
