import { renderHook } from '@testing-library/react'
import { expect, it } from 'vitest'
import { ApiError } from '../api/errors'
import { errorMessage, useFieldErrors } from './form-errors'

it('uses client copy for a validation detail', () => {
  const e = new ApiError(400, 'validation_failed', 'bad', [{ field: 'endsOn', message: 'must be after startsOn' }])
  const { result } = renderHook(() => useFieldErrors(e, { 'validation_failed:endsOn': 'End date must be after the start date.' }))
  expect(result.current).toEqual({ endsOn: 'End date must be after the start date.' })
})

it('sentence-cases the server message without copy', () => {
  const e = new ApiError(400, 'validation_failed', 'bad', [{ field: 'name', message: 'is required' }])
  const { result } = renderHook(() => useFieldErrors(e))
  expect(result.current).toEqual({ name: 'Is required.' })
})

it('places a detail-less 409 via copy keys', () => {
  const e = new ApiError(409, 'conflict', 'department code already exists', [])
  const { result } = renderHook(() => useFieldErrors(e, { 'conflict:code': 'A department with this code already exists.' }))
  expect(result.current).toEqual({ code: 'A department with this code already exists.' })
})

it('errorMessage covers rate limit, request id and plain errors', () => {
  expect(errorMessage(new ApiError(429, 'rate_limited', 'Too many requests, try again later', []))).toBe('Too many requests, try again later')
  expect(errorMessage(new ApiError(500, 'internal', 'x', [], 'abc'))).toContain('Reference: abc')
  expect(errorMessage(new Error('x'))).toBe('Something went wrong. Check your connection and try again.')
})
