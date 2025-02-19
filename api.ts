// API base URL
const API_BASE_URL = 'http://localhost:8080/api/v1';

// Tip tanımlamaları
interface RecipeSale {
  recipeId: number;
  quantity: number;
  saleDate: string;
  salePrice: number;
  customerName?: string;
  customerPhone?: string;
  note?: string;
  discount?: number;
  vat?: number;
}

interface Sale {
  id: number;
  productId: number;
  product: Product;
  quantity: number;
  saleDate: string;
  salePrice: number;
  discount: number;
  vat: number;
  netPrice: number;
  vatAmount: number;
  totalPrice: number;
  customerName: string;
  customerPhone: string;
  note?: string;
  unitCost: number;
  createdAt: string;
  updatedAt: string;
}

interface Product {
  id: number;
  companyName: string;
  category: string;
  productName: string;
  unit: string;
  invoiceNo: string;
  invoiceDate: string;
  initialStock: number;
  currentStock: number;
  unitPrice: number;
  vat: number;
  totalCost: number;
  createdAt: string;
  updatedAt: string;
}

interface SaleFormData {
  recipeId: number;
  quantity: number;
  saleDate: string;
  salePrice: number;
  customerName?: string;
  customerPhone?: string;
  note?: string;
  discount?: number;
  vat?: number;
}

interface Recipe {
    id: number;
    name: string;
    description?: string;
    outputQuantity: number;
    suggestedPrice: number;
    recipeItems: RecipeItem[];
}

interface RecipeItem {
    id: number;
    productId: number;
    quantity: number;
    description?: string;
    product: Product;
}

interface StockUsage {
    id: number;
    saleId: number;
    stockMovementId: number;
    usedQuantity: number;
}

interface RecipeSaleResponse {
    data: {
        sales: Sale[];
        recipe: Recipe;
        stockUsages: StockUsage[];
    };
}

export const createRecipeSale = async (recipeSale: RecipeSale): Promise<RecipeSaleResponse> => {
    const response = await fetch(`${API_BASE_URL}/sales/recipe`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(recipeSale),
    });

    if (!response.ok) {
        throw new Error('Reçete satışı oluşturulamadı');
    }

    return response.json();
}; 