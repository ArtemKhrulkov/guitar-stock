export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const backendUrl = config.public.apiBackendUrl || process.env.NUXT_PUBLIC_API_BACKEND_URL || 'http://localhost:8080'
  
  const path = event.path
  const query = getQuery(event)
  
  const fullUrl = `${backendUrl}${path}`
  
  const method = getMethod(event)
  const headers = getHeaders(event)
  
  const body = method !== 'GET' && method !== 'HEAD' 
    ? await readBody(event) 
    : undefined

  try {
    const response = await $fetch(fullUrl, {
      method,
      headers: {
        ...headers,
        host: undefined,
      },
      body,
      query,
    })
    
    return response
  } catch (error: any) {
    throw createError({
      statusCode: error.status || 500,
      statusMessage: error.statusText || 'Proxy Error',
      message: error.data?.message || error.message,
    })
  }
})
