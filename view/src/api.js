const API_BASE_URL = '/api/v1'

async function request(url, options = {}) {
  const response = await fetch(`${API_BASE_URL}${url}`, options)

  let data = null

  try {
    data = await response.json()
  } catch {
    data = null
  }

  if (!response.ok) {
    throw new Error(data?.error || 'Request failed')
  }

  return data
}

export async function getProducts() {
  return request('/products')
}

export async function searchProducts(query) {
  return request(`/products/search?q=${encodeURIComponent(query)}`)
}

export async function createProduct(product) {
  return request('/products', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(product)
  })
}

export async function updateProduct(id, product) {
  return request(`/products/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(product)
  })
}

export async function deleteProduct(id) {
  return request(`/products/${id}`, {
    method: 'DELETE'
  })
}

export async function importProducts(file) {
  const formData = new FormData()
  formData.append('file', file)

  return request('/products/import', {
    method: 'POST',
    body: formData
  })
}

export async function createPurchase(items, paymentFailure = false) {
  const headers = {
    'Content-Type': 'application/json',
    'Idempotency-Key': crypto.randomUUID()
  }

  if (paymentFailure) {
    headers['X-Fake-Payment'] = 'failure'
  }

  return request('/purchases', {
    method: 'POST',
    headers,
    body: JSON.stringify({
      items
    })
  })
}