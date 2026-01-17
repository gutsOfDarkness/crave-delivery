-- Seed data for Menu Items
TRUNCATE TABLE menu_items CASCADE;
INSERT INTO menu_items (name, description, price, category, image_url, is_available, created_at, updated_at) VALUES
('Chicken Biryani', 'Aromatic basmati rice cooked with tender chicken and spices.', 25000, 'Main Course', 'assets/images/chicken_biryani.png', true, NOW(), NOW()),
('Paneer Butter Masala', 'Soft paneer cubes in a rich, creamy tomato gravy.', 20000, 'Main Course', 'assets/images/paneer_butter_masala.png', true, NOW(), NOW()),
('Tandoori Chicken', 'Chicken marinated in yogurt and spices, roasted in a tandoor.', 30000, 'Starters', 'assets/images/tandoori_chicken.png', true, NOW(), NOW()),
('Butter Naan', 'Soft, fluffy indian bread topped with butter.', 4000, 'Breads', 'assets/images/butter_naan.png', true, NOW(), NOW()),
('Mango Lassi', 'Refreshing yogurt drink blended with sweet mango pulp.', 8000, 'Drinks', 'assets/images/mango_lassi.png', true, NOW(), NOW()),
('Gulab Jamun', 'Deep-fried milk solids soaked in sugar syrup.', 6000, 'Desserts', 'assets/images/gulab_jamun.png', true, NOW(), NOW());
