package gitmon

var removePatterns = []string{
	`/type="[A-Za-z0-9]+-text\/javascript"/`,
	`/<meta name="csrf-token" content=\"[A-Za-z0-9]+\">/`, 
	`/value=\"[^"]+\"/`,
	`/value='[^']+'/`, 
	// For Cloudflare
	`/token="[^"]+"/`, 
	// WordFence tokens ....
	`/hid=[A-Za-z0-9]+/`,
	// @todo for wpnonces
	// @todo csrf token patterns
};
