import 'menu_item.dart';

/// Cart item model representing a menu item with quantity in the user's cart.
class CartItem {
  final MenuItem menuItem;
  final int quantity;

  const CartItem({
    required this.menuItem,
    required this.quantity,
  });

  /// Subtotal in paisa
  int get subtotal => menuItem.price * quantity;

  /// Formatted subtotal string
  String get formattedSubtotal => '₹${(subtotal / 100.0).toStringAsFixed(2)}';

  CartItem copyWith({
    MenuItem? menuItem,
    int? quantity,
  }) {
    return CartItem(
      menuItem: menuItem ?? this.menuItem,
      quantity: quantity ?? this.quantity,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'menu_item_id': menuItem.id,
      'quantity': quantity,
    };
  }

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CartItem &&
          runtimeType == other.runtimeType &&
          menuItem.id == other.menuItem.id;

  @override
  int get hashCode => menuItem.id.hashCode;
}
