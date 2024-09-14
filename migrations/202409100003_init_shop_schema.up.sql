CREATE TYPE cart_status AS ENUM ('active', 'abandoned', 'converted');
CREATE TYPE order_status AS ENUM ('pending', 'processing', 'completed', 'cancelled', 'refunded', 'disputed', 'partially_refunded');
CREATE TYPE stock_movement_type AS ENUM ('in', 'out', 'reserve', 'release');
CREATE TYPE stock_movement_reference_type AS ENUM ('order', 'return', 'adjustment', 'cart');

CREATE TABLE categories (
                            id SERIAL PRIMARY KEY,
                            name VARCHAR(255) NOT NULL,
                            description VARCHAR(255),
                            parent_id INTEGER REFERENCES categories(id),
                            created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                            updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE product_categories (
                                    product_id VARCHAR(255) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
                                    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
                                    PRIMARY KEY (product_id, category_id),
                                    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


CREATE TABLE stocks (
                        id SERIAL PRIMARY KEY,
                        product_id VARCHAR(255) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
                        quantity INTEGER NOT NULL DEFAULT 0,
                        reserved_quantity INTEGER NOT NULL DEFAULT 0,
                        location VARCHAR(255),
                        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 庫存記錄表
CREATE TABLE stock_movements (
                                 id SERIAL PRIMARY KEY,
                                 stock_id INTEGER NOT NULL REFERENCES stocks(id),
                                 quantity INTEGER NOT NULL,
                                 type stock_movement_type NOT NULL,
                                 reference_id INTEGER,
                                 reference_type stock_movement_reference_type,
                                 created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


-- 購物車表
CREATE TABLE carts (
                       id SERIAL PRIMARY KEY,
                       customer_id VARCHAR(255) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                       status cart_status NOT NULL DEFAULT 'active',
                       currency currency NOT NULL,
                       subtotal DECIMAL(10, 2) NOT NULL DEFAULT 0,
                       tax DECIMAL(10, 2) NOT NULL DEFAULT 0,
                       discount DECIMAL(10, 2) NOT NULL DEFAULT 0,
                       total DECIMAL(10, 2) NOT NULL DEFAULT 0,
                       created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                       updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                       expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (NOW() + INTERVAL '7 days')
);

-- 購物車項目表
CREATE TABLE cart_items (
                            id SERIAL PRIMARY KEY,
                            cart_id INTEGER NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
                            product_id VARCHAR(255) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
                            price_id VARCHAR(255) NOT NULL REFERENCES prices(id) ON DELETE CASCADE,
                            stock_id INTEGER REFERENCES stocks(id),
                            quantity INTEGER NOT NULL CHECK (quantity > 0),
                            unit_price DECIMAL(10, 2) NOT NULL,
                            subtotal DECIMAL(10, 2) NOT NULL,
                            created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                            updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 訂單表
CREATE TABLE orders (
                        id SERIAL PRIMARY KEY,
                        customer_id VARCHAR(255) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
                        cart_id INTEGER REFERENCES carts(id) ON DELETE SET NULL,
                        status order_status NOT NULL DEFAULT 'pending',
                        currency currency NOT NULL,
                        subtotal DECIMAL(10, 2) NOT NULL,
                        tax DECIMAL(10, 2) NOT NULL DEFAULT 0,
                        discount DECIMAL(10, 2) NOT NULL DEFAULT 0,
                        total DECIMAL(10, 2) NOT NULL,
                        payment_intent_id VARCHAR(255) REFERENCES payment_intents(id),
                        invoice_id VARCHAR(255),
                        subscription_id VARCHAR(255),
                        refund_id VARCHAR(255),
                        shipping_address JSONB NOT NULL,
                        billing_address JSONB NOT NULL,
                        created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                        updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- 訂單項目表
CREATE TABLE order_items (
                             id SERIAL PRIMARY KEY,
                             order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
                             product_id VARCHAR(255) NOT NULL REFERENCES products(id) ON DELETE CASCADE,
                             price_id VARCHAR(255) NOT NULL REFERENCES prices(id) ON DELETE CASCADE,
                             stock_id INTEGER REFERENCES stocks(id),
                             quantity INTEGER NOT NULL CHECK (quantity > 0),
                             unit_price DECIMAL(10, 2) NOT NULL,
                             subtotal DECIMAL(10, 2) NOT NULL,
                             created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
                             updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
-- 索引
CREATE INDEX idx_carts_customer_id ON carts(customer_id);
CREATE INDEX idx_carts_status ON carts(status);
CREATE INDEX idx_cart_items_cart_id ON cart_items(cart_id);
CREATE INDEX idx_cart_items_product_id ON cart_items(product_id);
CREATE INDEX idx_orders_customer_id ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_refund_id ON orders(refund_id);
CREATE INDEX idx_orders_payment_intent_id ON orders(payment_intent_id);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_product_categories_product_id ON product_categories(product_id);
CREATE INDEX idx_product_categories_category_id ON product_categories(category_id);
CREATE INDEX idx_stocks_product_id ON stocks(product_id);
CREATE INDEX idx_stock_movements_stock_id ON stock_movements(stock_id);
CREATE INDEX idx_stock_movements_reference ON stock_movements(reference_type, reference_id);

