/// Menu item model representing a food item from the backend.
/// Price is stored in paisa (1/100 rupee) for precision.
class MenuItem {
  final String id;
  final String name;
  final String description;
  final int price; // Price in paisa
  final String category;
  final String? imageUrl;
  final bool isAvailable;

  const MenuItem({
    required this.id,
    required this.name,
    required this.description,
    required this.price,
    required this.category,
    this.imageUrl,
    this.isAvailable = true,
  });

  /// Price formatted in rupees for display
  double get priceInRupees => price / 100.0;

  /// Formatted price string (e.g., "₹150.00")
  String get formattedPrice => '₹${priceInRupees.toStringAsFixed(2)}';

  factory MenuItem.fromJson(Map<String, dynamic> json) {
    return MenuItem(
      id: json['id'] as String,
      name: json['name'] as String,
      description: json['description'] as String? ?? '',
      price: json['price'] as int,
      category: json['category'] as String,
      imageUrl: json['image_url'] as String?,
      isAvailable: json['is_available'] as bool? ?? true,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'description': description,
      'price': price,
      'category': category,
      'image_url': imageUrl,
      'is_available': isAvailable,
    };
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is MenuItem && runtimeType == other.runtimeType && id == other.id;

  @override
  int get hashCode => id.hashCode;
}
