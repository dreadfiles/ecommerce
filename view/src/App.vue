<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import {
  createProduct,
  createPurchase,
  deleteProduct,
  getProducts,
  importProducts,
  searchProducts,
  updateProduct
} from './api'

const products = ref([])
const searchQuery = ref('')
const loading = ref(false)
const error = ref('')
const success = ref('')
const showForm = ref(false)
const editingId = ref(null)
const selectedFile = ref(null)
const importing = ref(false)
const fileInput = ref(null)
const cart = ref([])
const purchasing = ref(false)
const paymentFailure = ref(false)
const purchaseResult = ref(null)
const notificationTimer = ref(null)
const showImport = ref(true)
const cartSection = ref(null)
const purchaseSection = ref(null)

const form = ref({
  name: '',
  sku: '',
  description: '',
  category: '',
  price: '',
  stock: '',
  weight_kg: ''
})

const cartCount = computed(() =>
  cart.value.reduce((total, item) => total + item.quantity, 0)
)

const cartTotal = computed(() =>
  cart.value.reduce(
    (total, item) => total + Number(item.price) * item.quantity,
    0
  )
)

function showSuccess(message) {
  error.value = ''
  success.value = message

  if (notificationTimer.value) {
    clearTimeout(notificationTimer.value)
  }

  notificationTimer.value = setTimeout(() => {
    success.value = ''
  }, 4000)
}

function showError(message) {
  success.value = ''
  error.value = message

  if (notificationTimer.value) {
    clearTimeout(notificationTimer.value)
  }

  notificationTimer.value = setTimeout(() => {
    error.value = ''
  }, 6000)
}

function resetForm() {
  form.value = {
    name: '',
    sku: '',
    description: '',
    category: '',
    price: '',
    stock: '',
    weight_kg: ''
  }

  editingId.value = null
}

function openCreateForm() {
  resetForm()
  showForm.value = true
  error.value = ''
  success.value = ''
}

function openEditForm(product) {
  editingId.value = product.id

  form.value = {
    name: product.name,
    sku: product.sku,
    description: product.description,
    category: product.category,
    price: product.price,
    stock: product.stock,
    weight_kg: product.weight_kg
  }

  showForm.value = true
  error.value = ''
  success.value = ''
}

function closeForm() {
  showForm.value = false
  resetForm()
}

function openImport() {
  showImport.value = true
}

function closeImport() {
  showImport.value = false
}

function clearImportFile() {
  selectedFile.value = null

  if (fileInput.value) {
    fileInput.value.value = ''
  }

  error.value = ''
  success.value = ''
}

function handleFileChange(event) {
  selectedFile.value = event.target.files[0] || null
  error.value = ''
  success.value = ''
}

async function handleImport() {
  if (!selectedFile.value) {
    showError('Please select a CSV file.')
    return
  }

  if (!selectedFile.value.name.toLowerCase().endsWith('.csv')) {
    showError('Please select a CSV file.')
    return
  }

  importing.value = true
  error.value = ''
  success.value = ''

  try {
    await importProducts(selectedFile.value)

    clearImportFile()

    await loadProducts()

    showSuccess('Products imported successfully.')
  } catch (err) {
    showError(err.message)
  } finally {
    importing.value = false
  }
}

async function loadProducts() {
  loading.value = true

  try {
    products.value = await getProducts()
  } catch (err) {
    showError(err.message)
  } finally {
    loading.value = false
  }
}

async function search() {
  const query = searchQuery.value.trim()

  if (!query) {
    await loadProducts()
    return
  }

  loading.value = true

  try {
    products.value = await searchProducts(query)
  } catch (err) {
    showError(err.message)
  } finally {
    loading.value = false
  }
}

async function clearSearch() {
  searchQuery.value = ''
  await loadProducts()
}

async function saveProduct() {
  error.value = ''
  success.value = ''

  const product = {
    name: form.value.name,
    sku: form.value.sku,
    description: form.value.description,
    category: form.value.category,
    price: String(form.value.price),
    stock: Number(form.value.stock),
    weight_kg: String(form.value.weight_kg)
  }

  try {
    if (editingId.value) {
      await updateProduct(editingId.value, product)
      closeForm()
      await loadProducts()
      showSuccess('Product updated successfully.')
    } else {
      await createProduct(product)
      closeForm()
      await loadProducts()
      showSuccess('Product created successfully.')
    }
  } catch (err) {
    showError(err.message)
  }
}

async function removeProduct(id) {
  if (!window.confirm('Are you sure you want to delete this product?')) {
    return
  }

  try {
    await deleteProduct(id)

    cart.value = cart.value.filter(item => item.id !== id)

    await loadProducts()

    showSuccess('Product deleted successfully.')
  } catch (err) {
    showError(err.message)
  }
}

function addToCart(product) {
  error.value = ''
  success.value = ''

  const existingItem = cart.value.find(item => item.id === product.id)

  if (existingItem) {
    if (existingItem.quantity >= product.stock) {
      showError('Quantity cannot exceed available stock.')
      return
    }

    existingItem.quantity += 1

    focusCart()

    return
  }

  if (product.stock <= 0) {
    showError('Product is out of stock.')
    return
  }

  cart.value.push({
    id: product.id,
    name: product.name,
    sku: product.sku,
    price: product.price,
    stock: product.stock,
    quantity: 1
  })

  showSuccess(`${product.name} added to cart.`)

  focusCart()
}

function increaseQuantity(item) {
  if (item.quantity >= item.stock) {
    showError('Quantity cannot exceed available stock.')
    return
  }

  item.quantity += 1
}

function decreaseQuantity(item) {
  if (item.quantity <= 1) {
    removeFromCart(item.id)
    return
  }

  item.quantity -= 1
}

function removeFromCart(id) {
  cart.value = cart.value.filter(item => item.id !== id)
}

function clearCart() {
  cart.value = []
  paymentFailure.value = false
}

async function focusCart() {
  await nextTick()

  if (cartSection.value) {
    cartSection.value.scrollIntoView({
      behavior: 'smooth',
      block: 'center'
    })
  }
}

async function focusPurchaseResult() {
  await nextTick()

  if (purchaseSection.value) {
    purchaseSection.value.scrollIntoView({
      behavior: 'smooth',
      block: 'center'
    })
  }
}

function closePurchaseResult() {
  purchaseResult.value = null
}

async function purchase() {
  if (cart.value.length === 0) {
    showError('The cart is empty.')
    return
  }

  purchasing.value = true
  error.value = ''
  success.value = ''
  purchaseResult.value = null

  const items = cart.value.map(item => ({
    product_id: item.id,
    quantity: item.quantity
  }))

  try {
    const result = await createPurchase(
      items,
      paymentFailure.value
    )

    purchaseResult.value = result

    cart.value = []

    paymentFailure.value = false

    await loadProducts()

    showSuccess(`Purchase #${result.id} completed successfully.`)

    await focusPurchaseResult()
  } catch (err) {
    showError(err.message)
  } finally {
    purchasing.value = false
  }
}

onMounted(loadProducts)
</script>

<template>
  <main class="app">
    <div v-if="success" class="notification success-notification">
      <div class="notification-icon">✓</div>

      <div class="notification-content">
        <strong>Success</strong>
        <span>{{ success }}</span>
      </div>

      <button
        class="notification-close"
        @click="success = ''"
      >
        ×
      </button>
    </div>

    <div v-if="error" class="notification error-notification">
      <div class="notification-icon">!</div>

      <div class="notification-content">
        <strong>Error</strong>
        <span>{{ error }}</span>
      </div>

      <button
        class="notification-close"
        @click="error = ''"
      >
        ×
      </button>
    </div>

    <header class="header">
      <div>
        <h1>E-Commerce</h1>
        <p>Product and purchase management</p>
      </div>

      <button
        class="cart-summary"
        :class="{ 'cart-summary-active': cartCount > 0 }"
        :disabled="cartCount === 0"
        @click="focusCart"
      >
        <span class="cart-icon">🛒</span>
        <span>Cart: {{ cartCount }} item{{ cartCount === 1 ? '' : 's' }}</span>
      </button>
    </header>

    <section class="content">
      <div class="page-header">
        <div>
          <h2>Products</h2>
          <p>Manage your products and create purchases.</p>
        </div>

        <div class="page-actions">
          <button @click="openCreateForm">
            New Product
          </button>
        </div>
      </div>

      <section v-if="showImport" class="import-section">
        <div class="import-header">
          <div class="import-title">
            <div class="import-icon">↥</div>

            <div>
              <h3>Import Products</h3>
              <p>Upload a CSV file with the product data.</p>
            </div>
          </div>

          <button
            class="icon-button"
            title="Close import"
            aria-label="Close import"
            @click="closeImport"
          >
            ×
          </button>
        </div>

        <div class="import-controls">
          <input
            ref="fileInput"
            type="file"
            accept=".csv,text/csv"
            @change="handleFileChange"
          />

          <button
            class="import-button"
            :disabled="importing"
            @click="handleImport"
          >
            {{ importing ? 'Importing...' : 'Import CSV' }}
          </button>

          <button
            v-if="selectedFile"
            class="text-button"
            :disabled="importing"
            @click="clearImportFile"
          >
            Clear file
          </button>
        </div>

        <div v-if="selectedFile" class="selected-file">
          <span class="file-check">✓</span>
          <span>{{ selectedFile.name }}</span>
        </div>
      </section>

      <div v-else class="import-closed">
        <div class="import-closed-info">
          <span class="import-closed-icon">↥</span>

          <div>
            <strong>Product import</strong>
            <span>CSV import</span>
          </div>
        </div>

        <button
          class="open-import-button"
          @click="openImport"
        >
          Open
        </button>
      </div>

      <div class="search">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search products..."
          @keyup.enter="search"
        />

        <button @click="search">
          Search
        </button>

        <button
          class="secondary"
          @click="clearSearch"
        >
          Clear
        </button>
      </div>

      <p v-if="loading" class="message">
        Loading products...
      </p>

      <p v-else-if="products.length === 0" class="message">
        No products found.
      </p>

      <div v-else class="table-container">
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>Name</th>
              <th>SKU</th>
              <th>Description</th>
              <th>Category</th>
              <th>Price</th>
              <th>Stock</th>
              <th>Weight (kg)</th>
              <th>Actions</th>
            </tr>
          </thead>

          <tbody>
            <tr
              v-for="product in products"
              :key="product.id"
            >
              <td>{{ product.id }}</td>
              <td>{{ product.name }}</td>
              <td>{{ product.sku }}</td>
              <td>{{ product.description }}</td>
              <td>{{ product.category }}</td>
              <td>{{ product.price }}</td>
              <td>{{ product.stock }}</td>
              <td>{{ product.weight_kg }}</td>

              <td class="actions">
                <button
                  class="small buy"
                  :disabled="product.stock <= 0"
                  @click="addToCart(product)"
                >
                  {{ product.stock > 0 ? 'Add to Cart' : 'Out of Stock' }}
                </button>

                <button
                  class="small"
                  @click="openEditForm(product)"
                >
                  Edit
                </button>

                <button
                  class="small danger"
                  @click="removeProduct(product.id)"
                >
                  Delete
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <section
        v-if="cart.length > 0"
        ref="cartSection"
        class="cart-section"
      >
        <div class="section-header">
          <div>
            <h2>Shopping Cart</h2>
            <p>{{ cartCount }} item{{ cartCount === 1 ? '' : 's' }}</p>
          </div>

          <button
            class="secondary"
            @click="clearCart"
          >
            Clear Cart
          </button>
        </div>

        <div class="cart-items">
          <div
            v-for="item in cart"
            :key="item.id"
            class="cart-item"
          >
            <div class="cart-product">
              <strong>{{ item.name }}</strong>
              <span>SKU: {{ item.sku }}</span>
              <span>Unit price: {{ item.price }}</span>
            </div>

            <div class="quantity-controls">
              <button
                class="quantity-button"
                @click="decreaseQuantity(item)"
              >
                -
              </button>

              <span>{{ item.quantity }}</span>

              <button
                class="quantity-button"
                @click="increaseQuantity(item)"
              >
                +
              </button>
            </div>

            <strong class="item-total">
              {{ (Number(item.price) * item.quantity).toFixed(2) }}
            </strong>

            <button
              class="small danger"
              @click="removeFromCart(item.id)"
            >
              Remove
            </button>
          </div>
        </div>

        <div class="checkout">
          <div class="checkout-total">
            <span>Total</span>
            <strong>{{ cartTotal.toFixed(2) }}</strong>
          </div>

          <label class="payment-option">
            <input
              v-model="paymentFailure"
              type="checkbox"
            />
            Simulate payment failure
          </label>

          <button
            class="purchase-button"
            :disabled="purchasing"
            @click="purchase"
          >
            {{ purchasing ? 'Processing...' : 'Complete Purchase' }}
          </button>
        </div>
      </section>

      <section
        v-if="purchaseResult"
        ref="purchaseSection"
        class="purchase-result"
      >
        <div class="section-header">
          <div>
            <div class="purchase-title">
              <span class="purchase-check">✓</span>

              <div>
                <h2>Purchase Completed</h2>
                <p>Purchase #{{ purchaseResult.id }}</p>
              </div>
            </div>
          </div>

          <div class="purchase-result-actions">
            <span class="status">
              {{ purchaseResult.status }}
            </span>

            <button
              class="icon-button result-close"
              title="Close purchase details"
              aria-label="Close purchase details"
              @click="closePurchaseResult"
            >
              ×
            </button>
          </div>
        </div>

        <div class="result-total">
          Total: <strong>{{ purchaseResult.total }}</strong>
        </div>

        <div class="result-items">
          <div
            v-for="item in purchaseResult.items"
            :key="item.product_id"
            class="result-item"
          >
            <span>Product #{{ item.product_id }}</span>
            <span>Quantity: {{ item.quantity }}</span>
            <span>Unit: {{ item.unit_price }}</span>
            <strong>{{ item.subtotal }}</strong>
          </div>
        </div>
      </section>
    </section>

    <div
      v-if="showForm"
      class="modal-backdrop"
    >
      <div class="modal">
        <div class="modal-header">
          <h2>
            {{ editingId ? 'Edit Product' : 'New Product' }}
          </h2>

          <button
            class="close"
            @click="closeForm"
          >
            ×
          </button>
        </div>

        <form @submit.prevent="saveProduct">
          <div class="form-grid">
            <label>
              Name
              <input
                v-model="form.name"
                type="text"
                required
              />
            </label>

            <label>
              SKU
              <input
                v-model="form.sku"
                type="text"
                required
              />
            </label>

            <label class="full-width">
              Description
              <input
                v-model="form.description"
                type="text"
                required
              />
            </label>

            <label>
              Category
              <input
                v-model="form.category"
                type="text"
                required
              />
            </label>

            <label>
              Price
              <input
                v-model="form.price"
                type="number"
                min="0"
                step="0.01"
                required
              />
            </label>

            <label>
              Stock
              <input
                v-model="form.stock"
                type="number"
                min="0"
                required
              />
            </label>

            <label>
              Weight (kg)
              <input
                v-model="form.weight_kg"
                type="number"
                min="0"
                step="0.01"
                required
              />
            </label>
          </div>

          <div class="form-actions">
            <button
              type="button"
              class="secondary"
              @click="closeForm"
            >
              Cancel
            </button>

            <button type="submit">
              {{ editingId ? 'Update' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>