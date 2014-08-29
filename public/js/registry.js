$(document).ready(
	function() {
		$('#search-box').keydown(function(event) { 
			if (event.keyCode == 13) {
				window.location.href = '/search?q=' + $('#search-box').val();
			}
		});
	}
)