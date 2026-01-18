-- Seed data for Menu Items
TRUNCATE TABLE menu_items CASCADE;
INSERT INTO menu_items (name, description, price, category, image_url, is_available, created_at, updated_at) VALUES
-- Breakfast Items
('Idli', 'Steamed rice cakes served with chutney and sambar', 4000, 'Breakfast', 'assets/images/idli.png', true, NOW(), NOW()),
('Dosa', 'Crispy rice crepe served with chutney and sambar', 5000, 'Breakfast', 'assets/images/dosa.png', true, NOW(), NOW()),
('Appe', 'Round fluffy dumplings made from fermented batter', 4500, 'Breakfast', 'assets/images/appe.png', true, NOW(), NOW()),
('Poori', 'Deep-fried puffed bread served with curry', 5000, 'Breakfast', 'assets/images/poori.png', true, NOW(), NOW()),
('Uttapam', 'Thick rice pancake with vegetables', 6000, 'Breakfast', 'assets/images/uttapam.png', true, NOW(), NOW()),
('Upma', 'Savory semolina porridge with vegetables', 4000, 'Breakfast', 'assets/images/upma.png', true, NOW(), NOW()),
('Omlette', 'Fluffy egg omlette with spices', 3000, 'Breakfast', 'assets/images/omlette.png', true, NOW(), NOW()),
('Vada', 'Crispy lentil fritters served with chutney', 3500, 'Breakfast', 'assets/images/vada.png', true, NOW(), NOW()),
('Bonda', 'Spiced potato balls deep-fried in chickpea batter', 3000, 'Breakfast', 'assets/images/bonda.png', true, NOW(), NOW()),
('Boiled Egg', 'Perfectly boiled eggs', 2000, 'Breakfast', 'assets/images/boiled_egg.png', true, NOW(), NOW()),
('Poha', 'Flattened rice with peanuts and spices', 3500, 'Breakfast', 'assets/images/poha.png', true, NOW(), NOW()),
('Tea', 'Hot brewed tea', 1500, 'Breakfast', 'assets/images/tea.png', true, NOW(), NOW()),
('Aloo Parantha', 'Stuffed potato flatbread', 5000, 'Breakfast', 'assets/images/aloo_parantha.png', true, NOW(), NOW()),

-- Fast Food Items
('Veg Noodles', 'Stir-fried noodles with vegetables', 7000, 'Fast Food', 'assets/images/veg_noodles.png', true, NOW(), NOW()),
('Gobi Noodles', 'Noodles with cauliflower', 7500, 'Fast Food', 'assets/images/gobi_noodles.png', true, NOW(), NOW()),
('Egg Noodles', 'Noodles with scrambled eggs', 8000, 'Fast Food', 'assets/images/egg_noodles.png', true, NOW(), NOW()),
('Veg Fried Rice', 'Fried rice with mixed vegetables', 7000, 'Fast Food', 'assets/images/veg_fried_rice.png', true, NOW(), NOW()),
('Gobi Fried Rice', 'Fried rice with cauliflower', 7500, 'Fast Food', 'assets/images/gobi_fried_rice.png', true, NOW(), NOW()),
('Egg Fried Rice', 'Fried rice with eggs', 8000, 'Fast Food', 'assets/images/egg_fried_rice.png', true, NOW(), NOW()),
('Mirchi Bajji', 'Spicy chili fritters', 4000, 'Fast Food', 'assets/images/mirchi_bajji.png', true, NOW(), NOW()),
('Aloo Bajji', 'Potato fritters', 3500, 'Fast Food', 'assets/images/aloo_bajji.png', true, NOW(), NOW()),
('Banana Bajji', 'Sweet banana fritters', 3000, 'Fast Food', 'assets/images/banana_bajji.png', true, NOW(), NOW()),
('Samosa', 'Crispy pastry filled with spiced potatoes', 2500, 'Fast Food', 'assets/images/samosa.png', true, NOW(), NOW()),
('Kachori', 'Spicy lentil stuffed fried bread', 2500, 'Fast Food', 'assets/images/kachori.png', true, NOW(), NOW()),
('Bread Pakoda', 'Bread slices dipped in gram flour batter and fried', 3000, 'Fast Food', 'assets/images/bread_pakoda.png', true, NOW(), NOW()),
('Pakoda', 'Mixed vegetable fritters', 3000, 'Fast Food', 'assets/images/pakoda.png', true, NOW(), NOW()),
('Gobi Manchurian', 'Cauliflower in spicy manchurian sauce', 8000, 'Fast Food', 'assets/images/gobi_manchurian.png', true, NOW(), NOW()),
('Bhel Puri', 'Puffed rice snack with tangy chutneys', 4000, 'Fast Food', 'assets/images/bhel_puri.png', true, NOW(), NOW()),
('Bread Omlette', 'Omlette served with bread', 4000, 'Fast Food', 'assets/images/bread_omlette.png', true, NOW(), NOW()),

-- Drinks
('Sweet Lemon Juice', 'Refreshing sweet lemon drink', 3000, 'Drinks', 'assets/images/sweet_lemon_juice.png', true, NOW(), NOW()),
('Salt Lemon Juice', 'Refreshing salted lemon drink', 3000, 'Drinks', 'assets/images/salt_lemon_juice.png', true, NOW(), NOW()),

-- Snacks
('Rose Flowers', 'Sweet rose-flavored treats', 2000, 'Snacks', 'assets/images/rose_flowers.png', true, NOW(), NOW()),
('Biscuits', 'Assorted biscuits', 1500, 'Snacks', 'assets/images/biscuits.png', true, NOW(), NOW()),
('Mixture', 'Savory snack mix', 2500, 'Snacks', 'assets/images/mixture.png', true, NOW(), NOW()),
('Aloo Fried Chips', 'Crispy potato chips', 2500, 'Snacks', 'assets/images/aloo_fried_chips.png', true, NOW(), NOW());