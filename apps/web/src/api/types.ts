// placeholder until P0-L4 runs npm run generate
export interface paths {
  '/me': {
    get: {
      responses: {
        200: { content: { 'application/json': { id: string; email: string } } }
      }
    }
  }
}
